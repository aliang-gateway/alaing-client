//go:build windows

package tray

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
	"unsafe"

	"aliang.one/nursorgate/common/logger"
	"aliang.one/nursorgate/common/version"
	"aliang.one/nursorgate/internal/ipc"
	"aliang.one/nursorgate/processor/setup"
	"github.com/getlantern/systray"
	"golang.org/x/sys/windows"
)

type windowsCompanionApp struct {
	mProxyStatus   *systray.MenuItem
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

func newWindowsCompanionApp() *windowsCompanionApp {
	writeWindowsCompanionTrace("newWindowsCompanionApp created")
	return &windowsCompanionApp{
		ipcClient: ipc.NewClient(),
		httpURL:   "http://127.0.0.1:56431",
		done:      make(chan struct{}),
	}
}

func (a *windowsCompanionApp) onReady() {
	writeWindowsCompanionTrace("onReady entered")
	logger.Info("Windows tray companion initialized")

	systray.SetIcon(GetIconDisabled())
	systray.SetTooltip("Aliang - Starting background service...")

	a.mOpenDashboard = systray.AddMenuItem("Open Dashboard", "Open the background service dashboard in browser")
	systray.AddSeparator()

	a.mProxyStatus = systray.AddMenuItem("Proxy: starting service...", "Current proxy listener status")
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
	a.mQuit = systray.AddMenuItem("Quit Aliang", "Quit the companion and stop the Windows service")

	go func() {
		a.connectAndStartHTTP()
		go a.handleMenuEvents()
		go a.syncStateLoop()
	}()
}

func (a *windowsCompanionApp) onExit() {
	logger.Info("Windows tray companion exiting")
}

func (a *windowsCompanionApp) connectAndStartHTTP() {
	serviceName := setup.GetServiceName()
	writeWindowsCompanionTrace("connectAndStartHTTP service=%s", serviceName)
	if !setup.IsServiceInstalled(serviceName, true) {
		writeWindowsCompanionTrace("service not installed")
		a.failStartup("background service not installed", "后台服务未安装，无法启动桌面程序。")
		return
	}

	status, err := setup.GetServiceStatus(serviceName, true)
	writeWindowsCompanionTrace("service status err=%v", err)
	if err == nil && !status.IsRunning {
		logger.Info("Windows service not running, starting it...")
		writeWindowsCompanionTrace("service not running, starting")
		if err := setup.StartService(serviceName, true); err != nil {
			logger.Error("Failed to start Windows service", "error", err)
			writeWindowsCompanionTrace("service start failed: %v", err)
			a.failStartup("background service start failed", fmt.Sprintf("后台服务启动失败：%v", err))
			return
		}
	}

	if !a.waitForIPC(10 * time.Second) {
		writeWindowsCompanionTrace("waitForIPC timed out")
		a.failStartup("background service startup timed out", "后台服务启动超时，未能建立 IPC 连接。")
		return
	}
	writeWindowsCompanionTrace("waitForIPC succeeded")

	resp, err := a.ipcClient.Send(ipc.ActionStartHTTP, nil)
	if err != nil {
		logger.Error("Failed to start dashboard via IPC", "error", err)
		writeWindowsCompanionTrace("ActionStartHTTP failed: %v", err)
		a.failStartup("failed to start dashboard", fmt.Sprintf("启动控制面板失败：%v", err))
		return
	}
	if !resp.OK {
		writeWindowsCompanionTrace("ActionStartHTTP rejected: %s", resp.Error)
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
	writeWindowsCompanionTrace("connectAndStartHTTP completed httpURL=%s", a.httpURL)
	go func() {
		time.Sleep(300 * time.Millisecond)
		a.openDashboard()
	}()
}

func (a *windowsCompanionApp) waitForIPC(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := a.ipcClient.Connect(); err == nil {
			writeWindowsCompanionTrace("ipc connect succeeded")
			return true
		}
		time.Sleep(400 * time.Millisecond)
	}
	return false
}

func (a *windowsCompanionApp) handleMenuEvents() {
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

func (a *windowsCompanionApp) startProxy() {
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

func (a *windowsCompanionApp) stopProxy() {
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

func (a *windowsCompanionApp) restartProxy() {
	a.stopProxy()
	time.Sleep(400 * time.Millisecond)
	a.startProxy()
}

func (a *windowsCompanionApp) syncStateLoop() {
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

func (a *windowsCompanionApp) syncState() {
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
	if running {
		systray.SetIcon(GetIcon())
		systray.SetTooltip("Aliang - Proxy Running")
	} else {
		systray.SetIcon(GetIconDisabled())
		systray.SetTooltip("Aliang - Proxy Stopped")
	}

	a.mProxyStatus.SetTitle(fmt.Sprintf("Proxy: %s", description))
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

func (a *windowsCompanionApp) applyUnavailableState(reason string) {
	systray.SetIcon(GetIconDisabled())
	systray.SetTooltip("Aliang - Service Unavailable")
	if a.mProxyStatus != nil {
		a.mProxyStatus.SetTitle(fmt.Sprintf("Proxy: %s", reason))
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
}

func (a *windowsCompanionApp) openDashboard() {
	cmd := newBackgroundCommand("rundll32", "url.dll,FileProtocolHandler", a.httpURL)
	if err := cmd.Start(); err != nil {
		logger.Error("Failed to open dashboard from Windows companion", "error", err)
	}
}

func (a *windowsCompanionApp) quit() {
	logger.Info("Quitting Windows tray companion...")

	select {
	case <-a.done:
	default:
		close(a.done)
	}

	a.stopProxy()
	_ = a.ipcClient.Close()
	_ = setup.StopService(setup.GetServiceName(), true)

	systray.Quit()
	os.Exit(0)
}

func RunWindowsCompanion() {
	writeWindowsCompanionTrace("RunWindowsCompanion entered")
	app := newWindowsCompanionApp()
	systray.Run(app.onReady, app.onExit)
}

func (a *windowsCompanionApp) failStartup(reason, message string) {
	a.applyUnavailableState(reason)
	showWindowsCompanionMessage("Aliang", message)
	writeWindowsCompanionTrace("failStartup reason=%s", reason)
	go func() {
		time.Sleep(200 * time.Millisecond)
		systray.Quit()
		os.Exit(1)
	}()
}

func showWindowsCompanionMessage(title, body string) {
	writeWindowsCompanionTrace("showWindowsCompanionMessage title=%s body=%s", title, body)
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
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer f.Close()

	line := fmt.Sprintf(format, args...)
	_, _ = fmt.Fprintf(f, "%s %s\n", time.Now().Format(time.RFC3339), line)
}
