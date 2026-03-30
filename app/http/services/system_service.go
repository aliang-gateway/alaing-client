package services

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"nursor.org/nursorgate/processor/setup"
)

type SystemServiceStatus struct {
	Supported      bool   `json:"supported"`
	Platform       string `json:"platform"`
	ServiceKind    string `json:"service_kind"`
	PlatformLabel  string `json:"platform_label"`
	ServiceLabel   string `json:"service_label"`
	DisplayName    string `json:"display_name"`
	ServiceName    string `json:"service_name"`
	Installed      bool   `json:"installed"`
	Running        bool   `json:"running"`
	Status         string `json:"status"`
	PID            int    `json:"pid"`
	RequiresAdmin  bool   `json:"requires_admin"`
	HasPrivileges  bool   `json:"has_privileges"`
	InstallScope   string `json:"install_scope"`
	ExecutablePath string `json:"executable_path,omitempty"`
	ConfigPath     string `json:"config_path,omitempty"`
	Message        string `json:"message,omitempty"`
	Warning        string `json:"warning,omitempty"`
}

type SystemServiceService struct{}

func NewSystemServiceService() *SystemServiceService {
	return &SystemServiceService{}
}

func (s *SystemServiceService) GetStatus() (SystemServiceStatus, error) {
	meta := buildSystemServiceStatusSkeleton()
	if !meta.Supported {
		meta.Status = "unsupported"
		meta.Message = "System service registration is not supported on this platform."
		return meta, nil
	}

	status, err := setup.GetServiceStatus(meta.ServiceName, true)
	if err != nil {
		return meta, fmt.Errorf("failed to inspect system service status: %w", err)
	}

	meta.Installed = status.IsInstalled
	meta.Running = status.IsRunning
	meta.Status = status.Status
	meta.PID = status.PID

	if !meta.Installed {
		meta.Message = fmt.Sprintf("%s is not registered yet.", meta.DisplayName)
	} else if meta.Running {
		meta.Message = fmt.Sprintf("%s is installed and currently running.", meta.DisplayName)
	} else {
		meta.Message = fmt.Sprintf("%s is installed but not running.", meta.DisplayName)
	}

	return meta, nil
}

func (s *SystemServiceService) Install() (SystemServiceStatus, error) {
	meta := buildSystemServiceStatusSkeleton()
	if !meta.Supported {
		meta.Status = "unsupported"
		meta.Message = "System service registration is not supported on this platform."
		return meta, nil
	}
	if !meta.HasPrivileges {
		return meta, setup.ErrNotRoot
	}

	options, err := buildSystemServiceInstallOptions()
	if err != nil {
		return meta, err
	}

	if err := setup.InstallService(options); err != nil {
		return meta, err
	}

	status, statusErr := s.GetStatus()
	if statusErr != nil {
		return meta, statusErr
	}
	status.Message = fmt.Sprintf("%s was registered successfully.", status.DisplayName)
	return status, nil
}

func (s *SystemServiceService) Uninstall() (SystemServiceStatus, error) {
	meta := buildSystemServiceStatusSkeleton()
	if !meta.Supported {
		meta.Status = "unsupported"
		meta.Message = "System service registration is not supported on this platform."
		return meta, nil
	}
	if !meta.HasPrivileges {
		return meta, setup.ErrNotRoot
	}

	if err := setup.UninstallService(meta.ServiceName, true); err != nil {
		return meta, err
	}

	status := buildSystemServiceStatusSkeleton()
	status.Status = "not_installed"
	status.Message = fmt.Sprintf("%s was uninstalled successfully.", status.DisplayName)
	return status, nil
}

func buildSystemServiceStatusSkeleton() SystemServiceStatus {
	serviceName := setup.GetServiceName()
	displayName := "Aliang Gateway System Service"
	serviceKind := ""
	platformLabel := ""
	serviceLabel := ""
	supported := true

	switch runtime.GOOS {
	case "windows":
		serviceKind = "windows_service"
		platformLabel = "Windows"
		serviceLabel = "Windows Service"
	case "darwin":
		serviceKind = "launch_daemon"
		platformLabel = "macOS"
		serviceLabel = "LaunchDaemon"
	case "linux":
		serviceKind = "systemd"
		platformLabel = "Linux"
		serviceLabel = "systemd"
	default:
		supported = false
		serviceKind = "unsupported"
		platformLabel = strings.Title(runtime.GOOS)
		serviceLabel = "Unsupported"
	}

	meta := SystemServiceStatus{
		Supported:     supported,
		Platform:      runtime.GOOS,
		ServiceKind:   serviceKind,
		PlatformLabel: platformLabel,
		ServiceLabel:  serviceLabel,
		DisplayName:   displayName,
		ServiceName:   serviceName,
		Status:        "unknown",
		RequiresAdmin: supported,
		HasPrivileges: setup.IsRoot(),
		InstallScope:  "system",
	}

	if executablePath, err := setup.GetCurrentExecutable(); err == nil {
		if managedPath, managedErr := resolveManagedExecutablePath(); managedErr == nil && managedPath != "" {
			meta.ExecutablePath = managedPath
		} else {
			meta.ExecutablePath = executablePath
		}
	}
	if configPath, err := resolveSystemServiceConfigPath(); err == nil {
		meta.ConfigPath = configPath
	}
	if runtime.GOOS == "windows" {
		meta.Warning = "Windows service registration is available, but the runtime service entrypoint should be verified in production."
	}

	return meta
}

func buildSystemServiceInstallOptions() (setup.InstallOptions, error) {
	currentExecPath, err := setup.GetCurrentExecutable()
	if err != nil {
		return setup.InstallOptions{}, fmt.Errorf("failed to get executable path: %w", err)
	}

	execPath, err := ensureManagedExecutable(currentExecPath)
	if err != nil {
		return setup.InstallOptions{}, fmt.Errorf("failed to prepare managed executable: %w", err)
	}

	configPath, err := resolveSystemServiceConfigPath()
	if err != nil {
		return setup.InstallOptions{}, err
	}

	workingDirectory := filepath.Dir(execPath)
	return setup.InstallOptions{
		Name:             setup.GetServiceName(),
		DisplayName:      "Aliang Gateway System Service",
		Description:      "Aliang background proxy service with automatic startup",
		ExecutablePath:   execPath,
		ConfigPath:       configPath,
		SystemWide:       true,
		StartType:        setup.StartAutomatic,
		Args:             []string{"start"},
		WorkingDirectory: workingDirectory,
	}, nil
}

func ensureManagedExecutable(sourcePath string) (string, error) {
	targetPath, err := resolveManagedExecutablePath()
	if err != nil {
		return "", err
	}
	if targetPath == "" {
		return sourcePath, nil
	}

	if sameFile(sourcePath, targetPath) {
		return targetPath, nil
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return "", err
	}
	if err := copyExecutableFile(sourcePath, targetPath); err != nil {
		return "", err
	}
	return targetPath, nil
}

func resolveManagedExecutablePath() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return "/Library/Application Support/Aliang/aliang", nil
	case "windows":
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, ".aliang", "aliang.exe"), nil
	default:
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, ".aliang", "aliang"), nil
	}
}

func copyExecutableFile(sourcePath string, targetPath string) error {
	src, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer src.Close()

	tempPath := targetPath + ".tmp"
	dst, err := os.OpenFile(tempPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}

	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return err
	}
	if err := dst.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tempPath, 0o755); err != nil && runtime.GOOS != "windows" {
		return err
	}
	if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(tempPath, targetPath)
}

func sameFile(left string, right string) bool {
	leftAbs, leftErr := filepath.Abs(left)
	rightAbs, rightErr := filepath.Abs(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return leftAbs == rightAbs
}

func resolveSystemServiceConfigPath() (string, error) {
	candidates := []string{}

	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, "config.new.json"))
		candidates = append(candidates, filepath.Join(cwd, "config.json"))
	}

	if homeDir, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(homeDir, ".aliang", "config.json"))
	}

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			absPath, absErr := filepath.Abs(candidate)
			if absErr == nil {
				return absPath, nil
			}
			return candidate, nil
		}
	}

	return "", nil
}
