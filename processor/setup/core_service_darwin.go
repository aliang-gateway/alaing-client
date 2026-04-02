//go:build darwin

package setup

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

const macOSCoreServiceLabel = "one.aliang.core"

// InstallMacOSCoreService installs the core service as a LaunchAgent.
// RunAtLoad=false, KeepAlive=false — the service is only started on demand via KickstartCoreService.
func InstallMacOSCoreService(execPath string) error {
	targetUser, err := resolveMacOSTrayAgentUser()
	if err != nil {
		return err
	}
	if strings.TrimSpace(execPath) == "" {
		execPath, err = GetCurrentExecutable()
		if err != nil {
			return fmt.Errorf("failed to determine executable for core service: %w", err)
		}
	}

	launchAgentsDir := filepath.Join(targetUser.HomeDir, "Library", "LaunchAgents")
	logDir := filepath.Join(targetUser.HomeDir, "Library", "Logs", "Aliang")
	plistPath := macOSCoreServicePlistPath(targetUser)

	if err := os.MkdirAll(launchAgentsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create LaunchAgents directory: %w", err)
	}
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return fmt.Errorf("failed to create core log directory: %w", err)
	}
	if err := chownPathToUserIfPossible(launchAgentsDir, targetUser); err != nil {
		return err
	}
	if err := chownPathToUserIfPossible(logDir, targetUser); err != nil {
		return err
	}

	plistContent, err := RenderLaunchdPlist(LaunchdPlistData{
		Label:             macOSCoreServiceLabel,
		ProgramPath:       execPath,
		Args:              []string{"start"},
		RunAtLoad:         false,
		KeepAlive:         false,
		WorkingDirectory:  filepath.Dir(execPath),
		StandardOutPath:   filepath.Join(logDir, "core.log"),
		StandardErrorPath: filepath.Join(logDir, "core.error.log"),
	})
	if err != nil {
		return fmt.Errorf("failed to render core service plist: %w", err)
	}

	if err := os.WriteFile(plistPath, []byte(plistContent), 0o644); err != nil {
		return fmt.Errorf("failed to write core service plist: %w", err)
	}
	if err := chownPathToUserIfPossible(plistPath, targetUser); err != nil {
		return err
	}

	// Bootstrap registers the service but does NOT start it (RunAtLoad=false)
	domain := fmt.Sprintf("gui/%s", targetUser.Uid)
	_, _ = exec.Command("launchctl", "bootout", domain, plistPath).CombinedOutput()

	if output, err := exec.Command("launchctl", "bootstrap", domain, plistPath).CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl bootstrap failed: %w, output: %s", err, strings.TrimSpace(string(output)))
	}

	return nil
}

// UninstallMacOSCoreService stops and removes the core service LaunchAgent.
func UninstallMacOSCoreService() error {
	targetUser, err := resolveMacOSTrayAgentUser()
	if err != nil {
		return err
	}

	plistPath := macOSCoreServicePlistPath(targetUser)
	if !FileExists(plistPath) {
		return nil
	}

	domain := fmt.Sprintf("gui/%s", targetUser.Uid)
	output, err := exec.Command("launchctl", "bootout", domain, plistPath).CombinedOutput()
	if err != nil {
		lowerOutput := strings.ToLower(string(output))
		if !strings.Contains(lowerOutput, "could not find service") &&
			!strings.Contains(lowerOutput, "service not found") &&
			!strings.Contains(lowerOutput, "no such process") {
			return fmt.Errorf("launchctl bootout failed: %w, output: %s", err, strings.TrimSpace(string(output)))
		}
	}

	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove core service plist: %w", err)
	}

	return nil
}

// KickstartCoreService starts the core service via launchctl kickstart.
func KickstartCoreService() error {
	targetUser, err := resolveMacOSTrayAgentUser()
	if err != nil {
		return err
	}

	domain := fmt.Sprintf("gui/%s", targetUser.Uid)
	target := fmt.Sprintf("%s/%s", domain, macOSCoreServiceLabel)
	if output, err := exec.Command("launchctl", "kickstart", target).CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl kickstart failed: %w, output: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// StopCoreServiceViaLaunchctl stops the core service via launchctl kill SIGTERM.
func StopCoreServiceViaLaunchctl() error {
	targetUser, err := resolveMacOSTrayAgentUser()
	if err != nil {
		return err
	}

	domain := fmt.Sprintf("gui/%s", targetUser.Uid)
	target := fmt.Sprintf("%s/%s", domain, macOSCoreServiceLabel)
	output, err := exec.Command("launchctl", "kill", "SIGTERM", target).CombinedOutput()
	if err != nil {
		lowerOutput := strings.ToLower(string(output))
		if strings.Contains(lowerOutput, "could not find service") ||
			strings.Contains(lowerOutput, "service not found") ||
			strings.Contains(lowerOutput, "unknown service") {
			return nil // already stopped
		}
		return fmt.Errorf("launchctl kill failed: %w, output: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// IsCoreServiceInstalled checks whether the core service plist exists.
func IsCoreServiceInstalled() bool {
	targetUser, err := resolveMacOSTrayAgentUser()
	if err != nil {
		return false
	}
	return FileExists(macOSCoreServicePlistPath(targetUser))
}

// IsCoreServiceRunning checks whether the core service is currently running.
func IsCoreServiceRunning() bool {
	targetUser, err := resolveMacOSTrayAgentUser()
	if err != nil {
		return false
	}
	domain := fmt.Sprintf("gui/%s", targetUser.Uid)
	output, err := exec.Command("launchctl", "print", fmt.Sprintf("%s/%s", domain, macOSCoreServiceLabel)).CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), "state = running")
}

func macOSCoreServicePlistPath(targetUser *user.User) string {
	return filepath.Join(targetUser.HomeDir, "Library", "LaunchAgents", macOSCoreServiceLabel+".plist")
}
