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

	applyMacOSTrayCompanionAdvisory(&meta)
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

	if runtime.GOOS == "darwin" {
		if err := setup.InstallMacOSTrayAgent(options.ExecutablePath); err != nil {
			meta.Warning = appendSystemServiceWarning(meta.Warning, fmt.Sprintf("System service was registered, but the macOS menu bar companion could not be installed: %v", err))
		}
	}

	status, statusErr := s.GetStatus()
	if statusErr != nil {
		return meta, statusErr
	}
	status.Message = fmt.Sprintf("%s was registered successfully.", status.DisplayName)
	if runtime.GOOS == "darwin" && status.Warning == "" {
		status.Message = fmt.Sprintf("%s was registered successfully. The macOS menu bar companion is configured and should show its icon in your login session.", status.DisplayName)
	}
	if meta.Warning != "" {
		status.Warning = appendSystemServiceWarning(status.Warning, meta.Warning)
	}
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
	if runtime.GOOS == "darwin" {
		if err := setup.UninstallMacOSTrayAgent(); err != nil {
			meta.Warning = appendSystemServiceWarning(meta.Warning, fmt.Sprintf("System service was removed, but the macOS menu bar companion could not be removed automatically: %v", err))
		}
	}
	if err := removeManagedExecutable(); err != nil {
		return meta, fmt.Errorf("service registration was removed, but failed to remove managed executable: %w", err)
	}

	status := buildSystemServiceStatusSkeleton()
	status.Status = "not_installed"
	status.Message = fmt.Sprintf("%s was uninstalled successfully. Managed executable removed, configuration file preserved.", status.DisplayName)
	if runtime.GOOS == "darwin" && meta.Warning == "" {
		status.Message = fmt.Sprintf("%s was uninstalled successfully. Managed executable removed, configuration file preserved, and the macOS menu bar companion was removed.", status.DisplayName)
	}
	if meta.Warning != "" {
		status.Warning = appendSystemServiceWarning(status.Warning, meta.Warning)
	}
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
	if configPath, err := ensureManagedSystemServiceConfigPath(); err == nil && configPath != "" {
		meta.ConfigPath = configPath
	}
	if runtime.GOOS == "windows" {
		meta.Warning = "Windows service registration is available, but the runtime service entrypoint should be verified in production."
	}

	return meta
}

func buildSystemServiceInstallOptions() (setup.InstallOptions, error) {
	return buildManagedSystemServiceInstallOptions("")
}

func BuildCLIServiceInstallOptions(requestedConfigPath string, systemWide bool) (setup.InstallOptions, error) {
	currentExecPath, err := setup.GetCurrentExecutable()
	if err != nil {
		return setup.InstallOptions{}, fmt.Errorf("failed to get executable path: %w", err)
	}

	if !systemWide {
		return setup.InstallOptions{
			Name:             setup.GetServiceName(),
			DisplayName:      "Aliang Gateway System Service",
			Description:      "Aliang background proxy service with automatic startup",
			ExecutablePath:   currentExecPath,
			ConfigPath:       strings.TrimSpace(requestedConfigPath),
			SystemWide:       false,
			StartType:        setup.StartAutomatic,
			WorkingDirectory: filepath.Dir(currentExecPath),
		}, nil
	}

	return buildManagedSystemServiceInstallOptions(requestedConfigPath)
}

func RemoveManagedSystemServiceExecutable() error {
	return removeManagedExecutable()
}

func buildManagedSystemServiceInstallOptions(preferredConfigPath string) (setup.InstallOptions, error) {
	currentExecPath, err := setup.GetCurrentExecutable()
	if err != nil {
		return setup.InstallOptions{}, fmt.Errorf("failed to get executable path: %w", err)
	}

	execPath, err := ensureManagedExecutable(currentExecPath)
	if err != nil {
		return setup.InstallOptions{}, fmt.Errorf("failed to prepare managed executable: %w", err)
	}

	configPath, err := ensureManagedSystemServiceConfigPathWithSource(preferredConfigPath)
	if err != nil {
		return setup.InstallOptions{}, fmt.Errorf("failed to prepare managed config path: %w", err)
	}

	workingDirectory := filepath.Dir(execPath)
	env, err := buildManagedSystemServiceEnvironment(configPath)
	if err != nil {
		return setup.InstallOptions{}, fmt.Errorf("failed to prepare managed runtime environment: %w", err)
	}
	return setup.InstallOptions{
		Name:             setup.GetServiceName(),
		DisplayName:      "Aliang Gateway System Service",
		Description:      "Aliang background proxy service with automatic startup",
		ExecutablePath:   execPath,
		ConfigPath:       configPath,
		SystemWide:       true,
		StartType:        setup.StartAutomatic,
		Args:             []string{"start"},
		Env:              env,
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

func removeManagedExecutable() error {
	targetPath, err := resolveManagedExecutablePath()
	if err != nil {
		return err
	}
	if targetPath == "" {
		return nil
	}
	if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func buildManagedSystemServiceEnvironment(configPath string) (map[string]string, error) {
	runtimeDataDir, err := resolveManagedRuntimeDataDir()
	if err != nil {
		return nil, err
	}

	env := map[string]string{}
	if runtimeDataDir != "" {
		env["NURSOR_CACHE_DIR"] = runtimeDataDir

		sourceDir := filepath.Dir(configPath)
		if err := ensureManagedCertificateAssets(sourceDir, runtimeDataDir); err != nil {
			return nil, err
		}
	}

	return env, nil
}

func resolveManagedRuntimeDataDir() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return "/Library/Application Support/Aliang", nil
	case "windows":
		programData := strings.TrimSpace(os.Getenv("ProgramData"))
		if programData == "" {
			return "", nil
		}
		return filepath.Join(programData, "Aliang"), nil
	case "linux":
		return "/var/lib/aliang", nil
	default:
		return "", nil
	}
}

func ensureManagedCertificateAssets(sourceDir string, targetDir string) error {
	sourceDir = strings.TrimSpace(sourceDir)
	targetDir = strings.TrimSpace(targetDir)
	if sourceDir == "" || targetDir == "" || sameFile(sourceDir, targetDir) {
		return nil
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return err
	}

	for _, name := range []string{
		"mitm-ca.pem",
		"mitm-ca.pem.key",
		"root-ca.pem",
		"root-ca.pem.key",
		"mtls-client.pem",
		"mtls-client.pem.key",
	} {
		sourcePath := filepath.Join(sourceDir, name)
		if _, err := os.Stat(sourcePath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}

		targetPath := filepath.Join(targetDir, name)
		if sameFile(sourcePath, targetPath) {
			continue
		}
		if err := copyRegularFile(sourcePath, targetPath, 0o600); err != nil {
			return err
		}
	}

	return nil
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
	if homeDir, err := os.UserHomeDir(); err == nil {
		canonicalPath := filepath.Join(homeDir, ".aliang", "config.json")
		if _, err := os.Stat(canonicalPath); err == nil {
			return canonicalPath, nil
		}
	}

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

func ensureManagedSystemServiceConfigPath() (string, error) {
	return ensureManagedSystemServiceConfigPathWithSource("")
}

func ensureManagedSystemServiceConfigPathWithSource(preferredSourcePath string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	targetPath := filepath.Join(homeDir, ".aliang", "config.json")
	if _, err := os.Stat(targetPath); err == nil {
		return targetPath, nil
	}

	sourcePath := strings.TrimSpace(preferredSourcePath)
	if sourcePath != "" {
		absSourcePath, absErr := filepath.Abs(sourcePath)
		if absErr != nil {
			return "", absErr
		}
		sourcePath = absSourcePath
	}
	if sourcePath == "" {
		resolvedPath, resolveErr := resolveSystemServiceConfigPath()
		if resolveErr != nil {
			return "", resolveErr
		}
		sourcePath = resolvedPath
	}
	if sourcePath == "" {
		return targetPath, nil
	}
	if sameFile(sourcePath, targetPath) {
		return targetPath, nil
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return "", err
	}
	if err := copyRegularFile(sourcePath, targetPath, 0o644); err != nil {
		return "", err
	}
	return targetPath, nil
}

func copyRegularFile(sourcePath string, targetPath string, mode os.FileMode) error {
	src, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer src.Close()

	tempPath := targetPath + ".tmp"
	dst, err := os.OpenFile(tempPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
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
	if err := os.Chmod(tempPath, mode); err != nil && runtime.GOOS != "windows" {
		return err
	}
	if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(tempPath, targetPath)
}

func applyMacOSTrayCompanionAdvisory(meta *SystemServiceStatus) {
	if meta == nil || runtime.GOOS != "darwin" {
		return
	}

	trayStatus, err := setup.GetMacOSTrayAgentStatus()
	if err != nil {
		meta.Warning = appendSystemServiceWarning(meta.Warning, fmt.Sprintf("Failed to inspect the macOS menu bar companion: %v", err))
		return
	}

	if !trayStatus.IsInstalled {
		if meta.Installed {
			meta.Warning = appendSystemServiceWarning(meta.Warning, "The background service is installed, but the macOS menu bar icon depends on a separate LaunchAgent companion that is not installed yet.")
		}
		return
	}

	if trayStatus.IsRunning {
		if meta.Installed && meta.Running {
			meta.Message = fmt.Sprintf("%s is installed and currently running. The macOS menu bar companion is active.", meta.DisplayName)
		}
		return
	}

	meta.Warning = appendSystemServiceWarning(meta.Warning, "The macOS menu bar companion is installed but not currently running in the logged-in GUI session.")
}

func appendSystemServiceWarning(existing string, addition string) string {
	addition = strings.TrimSpace(addition)
	if addition == "" {
		return existing
	}

	existing = strings.TrimSpace(existing)
	if existing == "" {
		return addition
	}
	return existing + " " + addition
}
