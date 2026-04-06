//go:build windows

package tray

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"aliang.one/nursorgate/common/logger"
	"aliang.one/nursorgate/common/version"
	"aliang.one/nursorgate/internal/ipc"
	"aliang.one/nursorgate/processor/setup"
	"github.com/getlantern/systray"
	"golang.org/x/sys/windows"
)

const maxCompanionTraceSize = 128 * 1024

type CompanionApp struct {
	mProxyStatus   *systray.MenuItem
	mModeStatus    *systray.MenuItem
	mModeHTTP      *systray.MenuItem
	mModeTUN       *systray.MenuItem
	mStart         *systray.MenuItem
	mStop          *systray.MenuItem
	mRestart       *systray.MenuItem
	mOpenDashboard *systray.MenuItem
	mQuit          *systray.MenuItem

	ipcClient *ipc.Client
	httpURL   string
	isRunning bool
	done      chan struct{}
}

func NewCompanionApp() *CompanionApp {
	return &CompanionApp{
		ipcClient: ipc.NewClient(),
		httpURL:   "http://127.0.0.1:56431",
		done:      make(chan struct{}),
	}
}

func (a *CompanionApp) onReady() {
	logger.Info("Windows tray companion initialized")

	systray.SetIcon(GetIconDisabled())
	systray.SetTooltip("Aliang - Starting background service...")

	a.mOpenDashboard = systray.AddMenuItem("Open Dashboard", "Open the background service dashboard in browser")
	systray.AddSeparator()

	a.mProxyStatus = systray.AddMenuItem("Proxy: starting service...", "Current proxy listener status")
	a.mProxyStatus.Disable()

	a.mModeStatus = systray.AddMenuItem("Current Mode: syncing...", "Selected proxy mode for the next start")
	a.mModeStatus.Disable()
	a.mModeHTTP = systray.AddMenuItemCheckbox("Select HTTP Mode", "Choose HTTP mode for the next explicit start", false)
	a.mModeTUN = systray.AddMenuItemCheckbox("Select TUN Mode", "Choose TUN mode for the next explicit start", false)

	systray.AddSeparator()

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
	a.mQuit = systray.AddMenuItem("Quit Aliang", "Quit the companion without stopping the background Windows service")

	go func() {
		a.connectAndStartHTTP()
		go a.handleMenuEvents()
		go a.syncStateLoop()
	}()
}

func (a *CompanionApp) onExit() {
	logger.Info("Windows tray companion exiting")
}

func (a *CompanionApp) connectAndStartHTTP() {
	serviceName := setup.GetServiceName()
	writeWindowsCompanionTrace("connectAndStartHTTP service=%s", serviceName)

	if a.waitForIPC(1500 * time.Millisecond) {
		logger.Info("Connected to existing Windows service via IPC")
	} else {
		status, err := setup.GetServiceStatus(serviceName, true)
		if err != nil {
			logger.Warn("Failed to query Windows service status before startup", "error", err)
		}

		serviceRunning := err == nil && status != nil && (status.IsRunning || status.Status == "start_pending")
		if !serviceRunning {
			logger.Info("Windows service not running, starting it...")
			if err := startServiceWithElevation(serviceName); err != nil {
				logger.Error("Failed to start Windows service", "error", err)
				a.failStartup("background service start failed", fmt.Sprintf("后台服务启动失败：%v", err))
				return
			}
		} else {
			logger.Info("Windows service already running, waiting for IPC...")
		}

		if !a.waitForIPC(15 * time.Second) {
			a.failStartup("background service startup timed out", "后台服务启动超时，未能建立 IPC 连接。")
			return
		}
	}

	resp, err := a.ipcClient.Send(ipc.ActionStartHTTP, nil)
	if err != nil {
		logger.Error("Failed to start dashboard via IPC", "error", err)
		a.failStartup("failed to start dashboard", fmt.Sprintf("启动控制面板失败：%v", err))
		return
	}
	if !resp.OK {
		a.failStartup("background service rejected dashboard start", "后台服务拒绝启动控制面板。")
		return
	}

	if data, ok := resp.Data.(map[string]interface{}); ok {
		if port, ok := data["port"].(string); ok && port != "" {
			a.httpURL = "http://127.0.0.1:" + port
		} else if portNum, ok := data["port"].(float64); ok {
			a.httpURL = fmt.Sprintf("http://127.0.0.1:%d", int(portNum))
		}
	}

	a.syncState()
	go func() {
		time.Sleep(300 * time.Millisecond)
		a.openDashboard()
	}()
}

func (a *CompanionApp) waitForIPC(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := a.ipcClient.Connect(); err == nil {
			return true
		}
		time.Sleep(400 * time.Millisecond)
	}
	return false
}

func (a *CompanionApp) handleMenuEvents() {
	for {
		select {
		case <-a.mStart.ClickedCh:
			a.startProxy()
		case <-a.mStop.ClickedCh:
			a.stopProxy()
		case <-a.mModeHTTP.ClickedCh:
			a.selectMode("http")
		case <-a.mModeTUN.ClickedCh:
			a.selectMode("tun")
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
	result, err := a.ipcClient.Send(ipc.ActionStartProxy, nil)
	if err != nil {
		logger.Error("Failed to start proxy from Windows companion", "error", err)
		a.applyUnavailableState("background service unavailable")
		return
	}
	if !result.OK {
		logger.Error("Background service rejected proxy start", "error", result.Error)
	}
	a.syncState()
}

func (a *CompanionApp) stopProxy() {
	result, err := a.ipcClient.Send(ipc.ActionStopProxy, nil)
	if err != nil {
		logger.Error("Failed to stop proxy from Windows companion", "error", err)
		a.applyUnavailableState("background service unavailable")
		return
	}
	if !result.OK && result.Error != "not_running" {
		logger.Error("Background service rejected proxy stop", "error", result.Error)
	}
	a.syncState()
}

func (a *CompanionApp) selectMode(mode string) {
	result, err := a.ipcClient.Send(ipc.ActionSwitchMode, ipc.SwitchModeArgs{Mode: mode})
	if err != nil {
		logger.Error("Failed to switch mode from Windows companion", "error", err)
		a.applyUnavailableState("background service unavailable")
		return
	}
	if !result.OK {
		logger.Error("Background service rejected mode switch", "error", result.Error)
	}
	a.syncState()
}

func (a *CompanionApp) restartProxy() {
	a.stopProxy()
	time.Sleep(400 * time.Millisecond)
	a.startProxy()
}

func (a *CompanionApp) syncModeMenu(mode string) {
	if a.mModeStatus != nil {
		a.mModeStatus.SetTitle(fmt.Sprintf("Current Mode: %s", strings.ToUpper(mode)))
	}
	if a.mModeHTTP != nil {
		if mode == "http" {
			a.mModeHTTP.Check()
		} else {
			a.mModeHTTP.Uncheck()
		}
	}
	if a.mModeTUN != nil {
		if mode == "tun" {
			a.mModeTUN.Check()
		} else {
			a.mModeTUN.Uncheck()
		}
	}
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
	result, err := a.ipcClient.Send(ipc.ActionGetStatus, nil)
	if err != nil {
		a.applyUnavailableState("background service unavailable")
		return
	}
	if !result.OK {
		a.applyUnavailableState("background service error")
		return
	}

	data, ok := result.Data.(map[string]interface{})
	if !ok {
		a.applyUnavailableState("invalid service state")
		return
	}

	mode := strings.ToLower(trayResultString(data, "current_mode"))
	if mode == "" {
		mode = "unknown"
	}
	running, _ := data["is_running"].(bool)
	description := trayResultString(data, "status")
	if description == "" {
		if running {
			description = "running"
		} else {
			description = "stopped"
		}
	}

	a.isRunning = running
	a.syncModeMenu(mode)
	if running {
		systray.SetIcon(GetIcon())
		systray.SetTooltip("Aliang - Proxy Running")
	} else {
		systray.SetIcon(GetIconDisabled())
		systray.SetTooltip("Aliang - Proxy Stopped")
	}

	a.mProxyStatus.SetTitle(fmt.Sprintf("Proxy: %s", description))
	a.mModeHTTP.Enable()
	a.mModeTUN.Enable()
	if running {
		a.mStart.Disable()
		a.mStop.Enable()
		a.mRestart.Enable()
	} else {
		a.mStart.Enable()
		a.mStop.Disable()
		a.mRestart.Disable()
	}
}

func (a *CompanionApp) applyUnavailableState(reason string) {
	systray.SetIcon(GetIconDisabled())
	systray.SetTooltip("Aliang - Service Unavailable")
	if a.mProxyStatus != nil {
		a.mProxyStatus.SetTitle(fmt.Sprintf("Proxy: %s", reason))
	}
	if a.mStart != nil {
		a.mStart.Disable()
	}
	if a.mModeHTTP != nil {
		a.mModeHTTP.Disable()
	}
	if a.mModeTUN != nil {
		a.mModeTUN.Disable()
	}
	if a.mStop != nil {
		a.mStop.Disable()
	}
	if a.mRestart != nil {
		a.mRestart.Disable()
	}
}

func (a *CompanionApp) openDashboard() {
	cmd := newBackgroundCommand("rundll32", "url.dll,FileProtocolHandler", a.httpURL)
	if err := cmd.Start(); err != nil {
		logger.Error("Failed to open dashboard from Windows companion", "error", err)
	}
}

func (a *CompanionApp) quit() {
	logger.Info("Quitting Windows tray companion...")

	select {
	case <-a.done:
	default:
		close(a.done)
	}

	_, _ = a.ipcClient.Send(ipc.ActionStopHTTP, nil)
	_ = a.ipcClient.Close()

	systray.Quit()
	os.Exit(0)
}

func RunCompanion() {
	app := NewCompanionApp()
	systray.Run(app.onReady, app.onExit)
}

func (a *CompanionApp) failStartup(reason, message string) {
	a.applyUnavailableState(reason)
	writeWindowsCompanionTrace("failStartup reason=%s", reason)
	showWindowsCompanionMessage("Aliang", message)
	go func() {
		time.Sleep(200 * time.Millisecond)
		systray.Quit()
		os.Exit(1)
	}()
}

func showWindowsCompanionMessage(title, body string) {
	user32 := windows.NewLazySystemDLL("user32.dll")
	messageBox := user32.NewProc("MessageBoxW")

	titlePtr, _ := windows.UTF16PtrFromString(title)
	bodyPtr, _ := windows.UTF16PtrFromString(body)

	const mbOK = 0x00000000
	const mbIconError = 0x00000010

	messageBox.Call(0, uintptr(unsafe.Pointer(bodyPtr)), uintptr(unsafe.Pointer(titlePtr)), mbOK|mbIconError)
}

func writeWindowsCompanionTrace(format string, args ...interface{}) {
	path := filepath.Join(os.TempDir(), "aliang-companion.log")
	if info, err := os.Stat(path); err == nil && info.Size() >= maxCompanionTraceSize {
		_ = os.Remove(path)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer f.Close()

	line := fmt.Sprintf(format, args...)
	_, _ = fmt.Fprintf(f, "%s %s\n", time.Now().Format(time.RFC3339), line)
}
