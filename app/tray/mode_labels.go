package tray

import "fmt"

func trayModeDisplayName(mode string) string {
	switch mode {
	case "http":
		return "Regular Mode"
	case "tun":
		return "Deep Mode"
	default:
		return "Unknown Mode"
	}
}

func trayCurrentModeTitle(mode string) string {
	return fmt.Sprintf("Current Mode: %s", trayModeDisplayName(mode))
}

func trayProxyTooltip(mode string, running bool) string {
	state := "Stopped"
	if running {
		state = "Running"
	}
	return fmt.Sprintf("Aliang - %s Proxy %s", trayModeDisplayName(mode), state)
}
