package tray

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	httpServer "aliang.one/nursorgate/app/http"
	"aliang.one/nursorgate/app/http/services"
	"aliang.one/nursorgate/common/logger"
	"aliang.one/nursorgate/common/version"
	startupRuntime "aliang.one/nursorgate/processor/runtime"
	"github.com/getlantern/systray"
)

// TrayApp manages the system tray application
type TrayApp struct {
	mProxyStatus   *systray.MenuItem
	mModeStatus    *systray.MenuItem
	mModeHTTP      *systray.MenuItem
	mModeTUN       *systray.MenuItem
	mStart         *systray.MenuItem
	mStop          *systray.MenuItem
	mRestart       *systray.MenuItem
	mOpenDashboard *systray.MenuItem
	mQuit          *systray.MenuItem

	isRunning  bool
	runService *services.RunService
	done       chan struct{}
	onStart    func()
	onStop     func()
	onRestart  func()
}

// NewTrayApp creates a new tray application instance
func NewTrayApp() *TrayApp {
	return &TrayApp{
		isRunning:  false,
		runService: services.GetSharedRunService(),
		done:       make(chan struct{}),
	}
}

// SetCallbacks sets the callback functions for menu actions
func (t *TrayApp) SetCallbacks(onStart, onStop, onRestart func()) {
	t.onStart = onStart
	t.onStop = onStop
	t.onRestart = onRestart
}

// onReady is called when the systray is ready
func onReady() {
	logger.Info("System tray initialized")

	// Set tray icon (inactive state initially)
	systray.SetIcon(GetIconDisabled())
	systray.SetTooltip("Aliang - Proxy Stopped")

	// Create menu items
	mOpenDashboard := systray.AddMenuItem("Open Dashboard", "Open web dashboard in browser")
	systray.AddSeparator()

	mProxyStatus := systray.AddMenuItem("Proxy: syncing status...", "Current proxy listener status")
	mProxyStatus.Disable()

	mModeStatus := systray.AddMenuItem("Current Mode: syncing...", "Selected proxy mode for the next start")
	mModeStatus.Disable()
	mModeHTTP := systray.AddMenuItemCheckbox("Select HTTP Mode", "Choose HTTP mode for the next explicit start", false)
	mModeTUN := systray.AddMenuItemCheckbox("Select TUN Mode", "Choose TUN mode for the next explicit start", false)

	systray.AddSeparator()

	mStart := systray.AddMenuItem("Start Proxy", "Start the active HTTP/TUN proxy listener")
	mStop := systray.AddMenuItem("Stop Proxy", "Stop the active HTTP/TUN proxy listener")
	mStop.Disable()
	mRestart := systray.AddMenuItem("Restart Proxy", "Restart the active HTTP/TUN proxy listener")
	mRestart.Disable()

	systray.AddSeparator()

	// Add version info
	versionInfo := fmt.Sprintf("Version: %s", version.String())
	mVersion := systray.AddMenuItem(versionInfo, "Application version")
	mVersion.Disable()

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Quit", "Quit the application")

	app := NewTrayApp()
	app.mProxyStatus = mProxyStatus
	app.mModeStatus = mModeStatus
	app.mModeHTTP = mModeHTTP
	app.mModeTUN = mModeTUN
	app.mStart = mStart
	app.mStop = mStop
	app.mRestart = mRestart
	app.mOpenDashboard = mOpenDashboard
	app.mQuit = mQuit

	// Set default callbacks
	app.SetCallbacks(
		func() { app.startProxy() },
		func() { app.stopProxy() },
		func() { app.restartProxy() },
	)

	// Ensure the dashboard/API server is available for local control UI.
	go func() {
		time.Sleep(250 * time.Millisecond)
		app.ensureDashboardServer()
		app.syncProxyState()
	}()

	// Handle menu clicks
	go app.handleMenuEvents()
	go app.syncProxyStateLoop()
}

// onExit is called when the systray is exiting
func onExit() {
	logger.Info("System tray exiting")
}

// handleMenuEvents handles all menu item click events
func (t *TrayApp) handleMenuEvents() {
	for {
		select {
		case <-t.mStart.ClickedCh:
			if t.onStart != nil {
				t.onStart()
			}

		case <-t.mStop.ClickedCh:
			if t.onStop != nil {
				t.onStop()
			}

		case <-t.mModeHTTP.ClickedCh:
			t.selectMode("http")

		case <-t.mModeTUN.ClickedCh:
			t.selectMode("tun")

		case <-t.mRestart.ClickedCh:
			if t.onRestart != nil {
				t.onRestart()
			}

		case <-t.mOpenDashboard.ClickedCh:
			t.openDashboard()

		case <-t.mQuit.ClickedCh:
			if t.quit() {
				return
			}
		}
	}
}

// startProxy starts the currently selected proxy mode.
func (t *TrayApp) startProxy() {
	if t.runService == nil {
		logger.Error("Tray run service is not initialized")
		return
	}

	logger.Info("Starting proxy from tray...")
	result := t.runService.StartService()
	status := trayResultString(result, "status")
	if status == "failed" {
		logger.Error(fmt.Sprintf("Failed to start proxy from tray: %s", trayResultMessage(result)))
		t.syncProxyState()
		return
	}
	logger.Info(fmt.Sprintf("Tray proxy start result: %s", trayResultMessage(result)))
	t.syncProxyState()
}

// stopProxy stops the currently active proxy mode.
func (t *TrayApp) stopProxy() {
	if t.runService == nil {
		logger.Error("Tray run service is not initialized")
		return
	}

	logger.Info("Stopping proxy from tray...")
	result := t.runService.StopService()
	status := trayResultString(result, "status")
	if status == "failed" && trayResultString(result, "error") != "not_running" {
		logger.Error(fmt.Sprintf("Failed to stop proxy from tray: %s", trayResultMessage(result)))
		t.syncProxyState()
		return
	}
	logger.Info(fmt.Sprintf("Tray proxy stop result: %s", trayResultMessage(result)))
	t.syncProxyState()
}

// restartProxy restarts the currently active proxy mode.
func (t *TrayApp) restartProxy() {
	logger.Info("Restarting proxy from tray...")
	t.stopProxy()
	time.Sleep(500 * time.Millisecond)
	t.startProxy()
}

func (t *TrayApp) selectMode(mode string) {
	if t.runService == nil {
		logger.Error("Tray run service is not initialized")
		return
	}

	logger.Info("Selecting " + mode + " mode from tray...")
	result := t.runService.SwitchMode(mode)
	if trayResultString(result, "status") == "failed" {
		logger.Error(fmt.Sprintf("Failed to switch tray mode: %s", trayResultMessage(result)))
		t.syncProxyState()
		return
	}

	logger.Info(fmt.Sprintf("Tray mode switch result: %s", trayResultMessage(result)))
	t.syncProxyState()
}

func (t *TrayApp) syncModeMenu(mode string) {
	if t.mModeStatus != nil {
		t.mModeStatus.SetTitle(fmt.Sprintf("Current Mode: %s", strings.ToUpper(mode)))
	}
	if t.mModeHTTP != nil {
		if mode == "http" {
			t.mModeHTTP.Check()
		} else {
			t.mModeHTTP.Uncheck()
		}
	}
	if t.mModeTUN != nil {
		if mode == "tun" {
			t.mModeTUN.Check()
		} else {
			t.mModeTUN.Uncheck()
		}
	}
}

// openDashboard opens the web dashboard in the default browser
func (t *TrayApp) openDashboard() {
	t.ensureDashboardServer()
	actualPort := t.waitForDashboardPort(3 * time.Second)
	if actualPort == "" {
		logger.Error("Failed to get dashboard port: HTTP server may not be ready")
		return
	}

	dashboardURL := fmt.Sprintf("http://localhost:%s", actualPort)

	var cmdName string
	var args []string
	switch runtime.GOOS {
	case "linux":
		cmdName = "xdg-open"
		args = []string{dashboardURL}
	case "windows":
		cmdName = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", dashboardURL}
	case "darwin":
		cmdName = "open"
		args = []string{dashboardURL}
	default:
		logger.Error(fmt.Sprintf("Unsupported platform: %s", runtime.GOOS))
		return
	}

	cmd := newBackgroundCommand(cmdName, args...)
	if err := cmd.Start(); err != nil {
		logger.Error(fmt.Sprintf("Failed to open dashboard: %v", err))
	}
}

// quit exits the application
func (t *TrayApp) quit() bool {
	logger.Info("Quitting application from tray...")

	if !t.stopProxyForQuit() {
		return false
	}

	select {
	case <-t.done:
	default:
		close(t.done)
	}

	if err := httpServer.StopHttpServer(); err != nil {
		logger.Error(fmt.Sprintf("Failed to stop dashboard server during quit: %v", err))
	}

	systray.Quit()
	os.Exit(0)
	return true
}

func (t *TrayApp) stopProxyForQuit() bool {
	if t.runService == nil {
		return true
	}

	logger.Info("Ensuring proxy is stopped before quit...")
	result := t.runService.StopService()
	if isAcceptableQuitProxyStopResult(result) {
		logger.Info(fmt.Sprintf("Tray quit proxy stop result: %s", trayResultMessage(result)))
		t.syncProxyState()
		return true
	}

	logger.Error(fmt.Sprintf("Failed to stop proxy during quit: %s", trayResultMessage(result)))
	t.syncProxyState()
	return false
}

func (t *TrayApp) ensureDashboardServer() {
	if httpServer.IsServerRunning() {
		return
	}

	logger.Info("Starting dashboard HTTP server for tray...")
	httpServer.StartHttpServer()
}

func (t *TrayApp) waitForDashboardPort(timeout time.Duration) string {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if actualPort := httpServer.GetActualPort(); actualPort != "" {
			return actualPort
		}
		time.Sleep(100 * time.Millisecond)
	}
	return httpServer.GetActualPort()
}

func (t *TrayApp) syncProxyStateLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.syncProxyState()
		case <-t.done:
			return
		}
	}
}

func (t *TrayApp) syncProxyState() {
	if t.runService == nil {
		return
	}

	status := t.runService.GetStatus()
	running, _ := status["is_running"].(bool)
	mode := strings.ToUpper(trayResultString(status, "current_mode"))
	if mode == "" {
		mode = "UNKNOWN"
	}
	description := trayResultString(status, "status")
	if description == "" {
		description = fmt.Sprintf("%s proxy stopped", mode)
	}

	t.isRunning = running
	t.syncModeMenu(strings.ToLower(mode))

	if t.mProxyStatus != nil {
		t.mProxyStatus.SetTitle(fmt.Sprintf("Proxy: %s", description))
	}

	if t.mStart != nil {
		if running {
			t.mStart.Disable()
		} else {
			t.mStart.Enable()
		}
	}
	if t.mStop != nil {
		if running {
			t.mStop.Enable()
		} else {
			t.mStop.Disable()
		}
	}
	if t.mRestart != nil {
		if running {
			t.mRestart.Enable()
		} else {
			t.mRestart.Disable()
		}
	}

	if running {
		systray.SetIcon(GetIcon())
		systray.SetTooltip(fmt.Sprintf("Aliang - %s Proxy Running", mode))
		return
	}

	systray.SetIcon(GetIconDisabled())
	startupStatus := startupRuntime.GetStartupState().GetStatus()
	if startupStatus == startupRuntime.READY || startupStatus == startupRuntime.CONFIGURED {
		systray.SetTooltip(fmt.Sprintf("Aliang - %s Proxy Stopped", mode))
		return
	}
	systray.SetTooltip(fmt.Sprintf("Aliang - %s Proxy Unavailable (%s)", mode, startupStatus))
}

func trayResultString(result map[string]interface{}, key string) string {
	if result == nil {
		return ""
	}
	value, _ := result[key].(string)
	return value
}

func trayResultMessage(result map[string]interface{}) string {
	for _, key := range []string{"msg", "message", "details", "status"} {
		if value := trayResultString(result, key); value != "" {
			return value
		}
	}
	return "unknown result"
}

func isAcceptableQuitProxyStopResult(result map[string]interface{}) bool {
	if result == nil {
		return false
	}

	status := trayResultString(result, "status")
	if status == "success" {
		return true
	}

	return status == "failed" && trayResultString(result, "error") == "not_running"
}

// Run starts the system tray application
// This is the main entry point for the tray application
func Run() {
	logger.Info("Starting system tray...")
	systray.Run(onReady, onExit)
}
