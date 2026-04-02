package cmd

import (
	"errors"
	"fmt"
	"net"
	"runtime"
	"syscall"

	"aliang.one/nursorgate/common/logger"
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
		logger.Error(fmt.Sprintf("Unsupported platform for opening dashboard: %s", runtime.GOOS))
		return
	}

	cmd := newBackgroundCommand(cmdName, args...)
	if err := cmd.Start(); err != nil {
		logger.Error(fmt.Sprintf("Failed to open dashboard: %v", err))
	}
}
