package tray

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/getlantern/systray"
	httpServer "nursor.org/nursorgate/app/http"
	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/common/version"
)

// TrayApp manages the system tray application
type TrayApp struct {
	mStart         *systray.MenuItem
	mStop          *systray.MenuItem
	mRestart       *systray.MenuItem
	mOpenDashboard *systray.MenuItem
	mQuit          *systray.MenuItem

	isRunning bool
	onStart   func()
	onStop    func()
	onRestart func()
}

// NewTrayApp creates a new tray application instance
func NewTrayApp() *TrayApp {
	return &TrayApp{
		isRunning: false,
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
	systray.SetTooltip("Nonelane - Stopped")

	// Create menu items
	mOpenDashboard := systray.AddMenuItem("Open Dashboard", "Open web dashboard in browser")
	systray.AddSeparator()

	mStart := systray.AddMenuItem("Start Server", "Start the HTTP server")
	mStop := systray.AddMenuItem("Stop Server", "Stop the HTTP server")
	mStop.Disable()
	mRestart := systray.AddMenuItem("Restart Server", "Restart the HTTP server")
	mRestart.Disable()

	systray.AddSeparator()

	// Add version info
	versionInfo := fmt.Sprintf("Version: %s", version.String())
	mVersion := systray.AddMenuItem(versionInfo, "Application version")
	mVersion.Disable()

	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Quit", "Quit the application")

	app := &TrayApp{
		mStart:         mStart,
		mStop:          mStop,
		mRestart:       mRestart,
		mOpenDashboard: mOpenDashboard,
		mQuit:          mQuit,
		isRunning:      false,
	}

	// Set default callbacks
	app.SetCallbacks(
		func() { app.startServer() },
		func() { app.stopServer() },
		func() { app.restartServer() },
	)

	// Start server automatically on tray ready
	go func() {
		time.Sleep(500 * time.Millisecond) // Small delay to ensure tray is ready
		app.startServer()
	}()

	// Handle menu clicks
	go app.handleMenuEvents()
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

		case <-t.mRestart.ClickedCh:
			if t.onRestart != nil {
				t.onRestart()
			}

		case <-t.mOpenDashboard.ClickedCh:
			t.openDashboard()

		case <-t.mQuit.ClickedCh:
			t.quit()
			return
		}
	}
}

// startServer starts the HTTP server
func (t *TrayApp) startServer() {
	if t.isRunning {
		logger.Info("Server is already running")
		return
	}

	logger.Info("Starting server from tray...")
	go httpServer.StartHttpServer()

	t.isRunning = true
	t.mStart.Disable()
	t.mStop.Enable()
	t.mRestart.Enable()

	// Update icon to indicate running state (colored)
	systray.SetIcon(GetIcon())
	systray.SetTooltip("Nonelane - Running")

	logger.Info("Server started successfully")
}

// stopServer stops the HTTP server
func (t *TrayApp) stopServer() {
	if !t.isRunning {
		logger.Info("Server is not running")
		return
	}

	logger.Info("Stopping server from tray...")
	if err := httpServer.StopHttpServer(); err != nil {
		logger.Error(fmt.Sprintf("Failed to stop server: %v", err))
		return
	}

	t.isRunning = false
	t.mStart.Enable()
	t.mStop.Disable()
	t.mRestart.Disable()

	// Update icon to indicate stopped state (gray)
	systray.SetIcon(GetIconDisabled())
	systray.SetTooltip("Nonelane - Stopped")

	logger.Info("Server stopped successfully")
}

// restartServer restarts the HTTP server
func (t *TrayApp) restartServer() {
	logger.Info("Restarting server from tray...")
	t.stopServer()
	time.Sleep(1 * time.Second) // Wait for server to stop
	t.startServer()
}

// openDashboard opens the web dashboard in the default browser
func (t *TrayApp) openDashboard() {
	dashboardURL := "http://localhost:8080" // Adjust to your actual port

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", dashboardURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", dashboardURL)
	case "darwin":
		cmd = exec.Command("open", dashboardURL)
	default:
		logger.Error(fmt.Sprintf("Unsupported platform: %s", runtime.GOOS))
		return
	}

	if err := cmd.Start(); err != nil {
		logger.Error(fmt.Sprintf("Failed to open dashboard: %v", err))
	}
}

// quit exits the application
func (t *TrayApp) quit() {
	logger.Info("Quitting application from tray...")

	if t.isRunning {
		t.stopServer()
	}

	systray.Quit()
	os.Exit(0)
}

// Run starts the system tray application
// This is the main entry point for the tray application
func Run() {
	logger.Info("Starting system tray...")
	systray.Run(onReady, onExit)
}
