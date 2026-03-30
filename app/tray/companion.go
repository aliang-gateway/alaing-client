package tray

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/getlantern/systray"
	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/common/version"
)

const companionDashboardBaseURL = "http://127.0.0.1:56431"

type companionControlClient struct {
	baseURL    string
	httpClient *http.Client
}

func newCompanionControlClient(baseURL string) *companionControlClient {
	return &companionControlClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 1500 * time.Millisecond,
		},
	}
}

func (c *companionControlClient) GetRunStatus() (map[string]interface{}, error) {
	return c.doJSON(http.MethodGet, "/api/run/status")
}

func (c *companionControlClient) StartProxy() (map[string]interface{}, error) {
	return c.doJSON(http.MethodPost, "/api/run/start")
}

func (c *companionControlClient) StopProxy() (map[string]interface{}, error) {
	return c.doJSON(http.MethodPost, "/api/run/stop")
}

func (c *companionControlClient) doJSON(method string, path string) (map[string]interface{}, error) {
	req, err := http.NewRequest(method, c.baseURL+path, bytes.NewReader(nil))
	if err != nil {
		return nil, err
	}
	if method != http.MethodGet {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var payload struct {
		Code    int                    `json:"code"`
		Msg     string                 `json:"msg"`
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 || payload.Code != 0 {
		message := payload.Msg
		if message == "" {
			message = payload.Message
		}
		if message == "" {
			message = fmt.Sprintf("request failed with status %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("%s", message)
	}

	if payload.Data == nil {
		return map[string]interface{}{}, nil
	}
	return payload.Data, nil
}

type CompanionApp struct {
	mProxyStatus   *systray.MenuItem
	mStart         *systray.MenuItem
	mStop          *systray.MenuItem
	mRestart       *systray.MenuItem
	mOpenDashboard *systray.MenuItem
	mQuit          *systray.MenuItem

	client    *companionControlClient
	isRunning bool
	done      chan struct{}
}

func NewCompanionApp() *CompanionApp {
	return &CompanionApp{
		client: newCompanionControlClient(companionDashboardBaseURL),
		done:   make(chan struct{}),
	}
}

func (a *CompanionApp) onReady() {
	logger.Info("macOS tray companion initialized")

	systray.SetIcon(GetIconDisabled())
	systray.SetTooltip("Aliang - Background Service Syncing")

	a.mOpenDashboard = systray.AddMenuItem("Open Dashboard", "Open the service dashboard in browser")
	systray.AddSeparator()

	a.mProxyStatus = systray.AddMenuItem("Proxy: syncing status...", "Current proxy listener status")
	a.mProxyStatus.Disable()

	a.mStart = systray.AddMenuItem("Start Proxy", "Start the active proxy listener in the background service")
	a.mStop = systray.AddMenuItem("Stop Proxy", "Stop the active proxy listener in the background service")
	a.mStop.Disable()
	a.mRestart = systray.AddMenuItem("Restart Proxy", "Restart the active proxy listener in the background service")
	a.mRestart.Disable()

	systray.AddSeparator()
	versionInfo := fmt.Sprintf("Version: %s", version.String())
	mVersion := systray.AddMenuItem(versionInfo, "Application version")
	mVersion.Disable()

	systray.AddSeparator()
	a.mQuit = systray.AddMenuItem("Quit Tray", "Quit the menu bar companion")

	go a.handleMenuEvents()
	go a.syncStateLoop()
	go a.syncState()
}

func (a *CompanionApp) onExit() {
	logger.Info("macOS tray companion exiting")
}

func (a *CompanionApp) handleMenuEvents() {
	for {
		select {
		case <-a.mStart.ClickedCh:
			a.startProxy()
		case <-a.mStop.ClickedCh:
			a.stopProxy()
		case <-a.mRestart.ClickedCh:
			a.restartProxy()
		case <-a.mOpenDashboard.ClickedCh:
			a.openDashboard()
		case <-a.mQuit.ClickedCh:
			a.quit()
			return
		case <-a.done:
			return
		}
	}
}

func (a *CompanionApp) startProxy() {
	logger.Info("Starting proxy from tray companion...")
	result, err := a.client.StartProxy()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to start proxy from tray companion: %v", err))
		a.applyUnavailableState(fmt.Sprintf("service unavailable (%v)", err))
		return
	}

	if trayResultString(result, "status") == "failed" {
		logger.Error(fmt.Sprintf("Background service rejected tray start request: %s", trayResultMessage(result)))
	}
	a.syncState()
}

func (a *CompanionApp) stopProxy() {
	logger.Info("Stopping proxy from tray companion...")
	result, err := a.client.StopProxy()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to stop proxy from tray companion: %v", err))
		a.applyUnavailableState(fmt.Sprintf("service unavailable (%v)", err))
		return
	}

	if status := trayResultString(result, "status"); status == "failed" && trayResultString(result, "error") != "not_running" {
		logger.Error(fmt.Sprintf("Background service rejected tray stop request: %s", trayResultMessage(result)))
	}
	a.syncState()
}

func (a *CompanionApp) restartProxy() {
	a.stopProxy()
	time.Sleep(400 * time.Millisecond)
	a.startProxy()
}

func (a *CompanionApp) syncStateLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.syncState()
		case <-a.done:
			return
		}
	}
}

func (a *CompanionApp) syncState() {
	status, err := a.client.GetRunStatus()
	if err != nil {
		a.applyUnavailableState("background service unavailable")
		return
	}

	running, _ := status["is_running"].(bool)
	mode := strings.ToUpper(trayResultString(status, "current_mode"))
	if mode == "" {
		mode = "UNKNOWN"
	}

	description := trayResultString(status, "status")
	if description == "" {
		if running {
			description = fmt.Sprintf("%s proxy running", mode)
		} else {
			description = fmt.Sprintf("%s proxy stopped", mode)
		}
	}

	a.isRunning = running
	if a.mProxyStatus != nil {
		a.mProxyStatus.SetTitle(fmt.Sprintf("Proxy: %s", description))
	}
	if a.mStart != nil {
		if running {
			a.mStart.Disable()
		} else {
			a.mStart.Enable()
		}
	}
	if a.mStop != nil {
		if running {
			a.mStop.Enable()
		} else {
			a.mStop.Disable()
		}
	}
	if a.mRestart != nil {
		if running {
			a.mRestart.Enable()
		} else {
			a.mRestart.Disable()
		}
	}

	if running {
		systray.SetIcon(GetIcon())
		systray.SetTooltip(fmt.Sprintf("Aliang - %s Proxy Running", mode))
		return
	}

	systray.SetIcon(GetIconDisabled())
	systray.SetTooltip(fmt.Sprintf("Aliang - %s Proxy Stopped", mode))
}

func (a *CompanionApp) applyUnavailableState(reason string) {
	a.isRunning = false
	if a.mProxyStatus != nil {
		a.mProxyStatus.SetTitle("Proxy: background service unavailable")
	}
	if a.mStart != nil {
		a.mStart.Disable()
	}
	if a.mStop != nil {
		a.mStop.Disable()
	}
	if a.mRestart != nil {
		a.mRestart.Disable()
	}

	systray.SetIcon(GetIconDisabled())
	systray.SetTooltip(fmt.Sprintf("Aliang - %s", reason))
}

func (a *CompanionApp) openDashboard() {
	var cmdName string
	var args []string
	switch runtime.GOOS {
	case "linux":
		cmdName = "xdg-open"
		args = []string{companionDashboardBaseURL}
	case "windows":
		cmdName = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", companionDashboardBaseURL}
	case "darwin":
		cmdName = "open"
		args = []string{companionDashboardBaseURL}
	default:
		logger.Error(fmt.Sprintf("Unsupported platform: %s", runtime.GOOS))
		return
	}

	cmd := newBackgroundCommand(cmdName, args...)
	if err := cmd.Start(); err != nil {
		logger.Error(fmt.Sprintf("Failed to open dashboard from tray companion: %v", err))
	}
}

func (a *CompanionApp) quit() {
	select {
	case <-a.done:
	default:
		close(a.done)
	}
	systray.Quit()
	os.Exit(0)
}

func RunCompanion() {
	app := NewCompanionApp()
	logger.Info("Starting macOS tray companion...")
	systray.Run(app.onReady, app.onExit)
}
