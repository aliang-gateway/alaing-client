package cmd

import (
	"errors"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"syscall"

	"nursor.org/nursorgate/common/logger"
)

const (
	singleInstanceGuardAddr = "127.0.0.1:56430"
	dashboardURL            = "http://localhost:56431"
)

func acquireSingleInstanceGuard() (net.Listener, bool, error) {
	listener, err := net.Listen("tcp", singleInstanceGuardAddr)
	if err != nil {
		if errors.Is(err, syscall.EADDRINUSE) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("failed to acquire single-instance guard on %s: %w", singleInstanceGuardAddr, err)
	}

	return listener, true, nil
}

func openDashboardInBrowser() {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", dashboardURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", dashboardURL)
	case "darwin":
		cmd = exec.Command("open", dashboardURL)
	default:
		logger.Error(fmt.Sprintf("Unsupported platform for opening dashboard: %s", runtime.GOOS))
		return
	}

	if err := cmd.Start(); err != nil {
		logger.Error(fmt.Sprintf("Failed to open dashboard: %v", err))
	}
}
