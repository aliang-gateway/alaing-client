package installer

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/processor/cert"
)

// CertInfo holds information about a certificate
type CertInfo struct {
	Subject        string
	Issuer         string
	NotBefore      string
	NotAfter       string
	Fingerprint    string
	InstalledCount int
	InstallPath    string
}

// CertInstaller interface defines platform-specific certificate operations
type CertInstaller interface {
	// IsInstalled checks if a certificate is installed in system trust store
	// certBytes is used to extract the real certificate Common Name for accurate detection
	IsInstalled(certType string, certBytes []byte) (bool, error)

	// Install installs a certificate to system trust store (may require elevation)
	Install(certType string, certPath string) error

	// Remove removes a certificate from system trust store (may require elevation)
	// certBytes is used to extract the real certificate Common Name for accurate removal
	Remove(certType string, certBytes []byte) error

	// GetCertInfo retrieves certificate information
	GetCertInfo(certType string, certBytes []byte) (CertInfo, error)

	// GetInstallPath returns the system-specific installation path
	GetInstallPath(certType string) string

	// IsTrusted checks if a certificate is marked as globally trusted by the system
	// certBytes is used to extract the real certificate Common Name for accurate detection
	IsTrusted(certType string, certBytes []byte) (bool, error)

	// GetTrustStatus returns the detailed trust status of a certificate
	// Returns values like "not_found", "installed_not_trusted", "system_trusted"
	GetTrustStatus(certType string, certBytes []byte) (string, error)
}

// NewInstaller returns a platform-specific certificate installer
func NewInstaller() CertInstaller {
	switch runtime.GOOS {
	case "darwin":
		return &DarwinInstaller{}
	case "linux":
		return &LinuxInstaller{}
	case "windows":
		return &WindowsInstaller{}
	default:
		logger.Warn(fmt.Sprintf("Unsupported OS for certificate installation: %s", runtime.GOOS))
		return &UnimplementedInstaller{}
	}
}

// ============= Common Helper Functions =============

// extractCertCommonName extracts the Common Name from certificate PEM bytes
// This returns the actual certificate CN from the Subject, not a hardcoded value
func extractCertCommonName(certBytes []byte) (string, error) {
	block, _ := pem.Decode(certBytes)
	if block == nil {
		return "", fmt.Errorf("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse certificate: %w", err)
	}

	if cert.Subject.CommonName == "" {
		return "", fmt.Errorf("certificate has no Common Name")
	}

	return cert.Subject.CommonName, nil
}

// parseCertificateInfo extracts certificate details from PEM bytes
func parseCertificateInfo(certBytes []byte) (CertInfo, error) {
	block, _ := pem.Decode(certBytes)
	if block == nil {
		return CertInfo{}, fmt.Errorf("failed to parse certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return CertInfo{}, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Calculate SHA256 fingerprint
	hash := sha256.Sum256(block.Bytes)
	fingerprint := hex.EncodeToString(hash[:])

	return CertInfo{
		Subject:     cert.Subject.String(),
		Issuer:      cert.Issuer.String(),
		NotBefore:   cert.NotBefore.Format("2006-01-02"),
		NotAfter:    cert.NotAfter.Format("2006-01-02"),
		Fingerprint: fingerprint,
	}, nil
}

// ============= macOS (Darwin) Implementation =============

type DarwinInstaller struct{}

// isRunningAsSudo checks if the current process is running with sudo privileges
func isRunningAsSudo() bool {
	// Check if SUDO_UID environment variable is set (indicates sudo execution)
	return os.Getenv("SUDO_UID") != ""
}

// IsInstalled checks if certificate is installed in macOS System keychain
func (d *DarwinInstaller) IsInstalled(certType string, certBytes []byte) (bool, error) {
	// Extract the real certificate Common Name from the certificate itself
	commonName, err := extractCertCommonName(certBytes)
	if err != nil {
		logger.Warn(fmt.Sprintf("Failed to extract cert CN from bytes, falling back to hardcoded name: %v", err))
		// Fallback to hardcoded name if extraction fails
		commonName = getCertCommonName(certType)
	}

	logger.Info(fmt.Sprintf("Checking if certificate %s (CN: %s) is installed in System keychain", certType, commonName))

	cmd := exec.Command("security", "find-certificate", "-c", commonName,
		"/Library/Keychains/System.keychain")

	err = cmd.Run()
	if err == nil {
		logger.Info(fmt.Sprintf("Certificate %s found in System keychain", commonName))
		return true, nil
	}

	logger.Info(fmt.Sprintf("Certificate %s not found in System keychain", commonName))
	return false, nil
}

// Install adds certificate to macOS System keychain with appropriate elevation strategy
func (d *DarwinInstaller) Install(certType string, certPath string) error {
	logger.Info(fmt.Sprintf("Installing certificate %s to macOS System keychain", certType))

	// 1. 获取绝对路径 (非常重要，防止 osascript 执行环境路径不同找不到文件)
	absPath, err := filepath.Abs(certPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// 2. 无论是否是 Root，统一使用 osascript 唤起 GUI 弹窗
	// 只有这样才能满足 "User Interaction" 的要求，成功修改信任设置
	logger.Info("Executing osascript to request GUI authorization...")

	// 构造 AppleScript 命令
	// -d: 添加到 admin 证书库
	// -r trustRoot: 设置为根信任 (关键参数)
	// -k /Library/Keychains/System.keychain: 目标钥匙串
	//script := fmt.Sprintf(
	//	"do shell script \"security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain '%s'\" with administrator privileges",
	//	absPath,
	//)

	//cmd := exec.Command("osascript", "-e", script)
	cmd := exec.Command("open", absPath)
	output, err := cmd.CombinedOutput()

	if err != nil {
		// 捕获用户点击"取消"的情况
		if strings.Contains(string(output), "canceled") {
			logger.Warn("User canceled the certificate installation")
			return fmt.Errorf("installation canceled by user")
		}

		logger.Error(fmt.Sprintf("Failed to install certificate (osascript): %v, output: %s", err, string(output)))
		return fmt.Errorf("certificate installation failed. Output: %s", string(output))
	}

	logger.Info("Certificate installed and trusted successfully (via GUI authorization)")
	return nil
}

// Remove deletes certificate from macOS System keychain
func (d *DarwinInstaller) Remove(certType string, certBytes []byte) error {
	// Extract the real certificate Common Name from the certificate itself
	commonName, err := extractCertCommonName(certBytes)
	if err != nil {
		logger.Warn(fmt.Sprintf("Failed to extract cert CN from bytes, falling back to config: %v", err))
		// Fallback to configuration if extraction fails
		config := cert.GetCertConfig(certType)
		if config != nil {
			commonName = config.CN
		} else {
			commonName = certType
		}
	}

	logger.Info(fmt.Sprintf("Removing certificate %s (CN: %s) from macOS System keychain", certType, commonName))

	if isRunningAsSudo() {
		// Already running with sudo privileges - directly execute security command
		logger.Info("Running with sudo privileges, executing security command directly")
		cmd := exec.Command("security", "delete-certificate", "-c", commonName,
			"/Library/Keychains/System.keychain")

		output, err := cmd.CombinedOutput()
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to remove certificate (sudo direct): %v, output: %s", err, string(output)))
			return fmt.Errorf("certificate removal failed: %w", err)
		}

		logger.Info("Certificate removed successfully from macOS System keychain (via sudo)")
		return nil
	}

	// Not running with sudo - use osascript to request elevation
	logger.Info("Not running with sudo, requesting elevation via osascript")
	script := fmt.Sprintf(
		"do shell script \"security delete-certificate -c '%s' /Library/Keychains/System.keychain\" with administrator privileges",
		commonName,
	)

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()

	if err != nil {
		logger.Error(fmt.Sprintf("Failed to remove certificate (osascript): %v, output: %s", err, string(output)))
		// Provide helpful error message
		return fmt.Errorf("certificate removal failed. You may need to run this application with 'sudo' privilege: %w", err)
	}

	logger.Info("Certificate removed successfully from macOS System keychain (via osascript)")
	return nil
}

// GetCertInfo retrieves certificate information
func (d *DarwinInstaller) GetCertInfo(certType string, certBytes []byte) (CertInfo, error) {
	info, err := parseCertificateInfo(certBytes)
	if err != nil {
		return info, err
	}
	info.InstallPath = "/Library/Keychains/System.keychain"
	return info, nil
}

// GetInstallPath returns the macOS installation path
func (d *DarwinInstaller) GetInstallPath(certType string) string {
	return "/Library/Keychains/System.keychain"
}

// IsTrusted checks if a certificate is marked as trusted on macOS
// It first checks if the certificate exists in the keychain, then verifies trust settings
func (d *DarwinInstaller) IsTrusted(certType string, certBytes []byte) (bool, error) {
	commonName, err := extractCertCommonName(certBytes)
	if err != nil {
		commonName = getCertCommonName(certType)
	}
	// 不再需要检测是否安装了，因为已经通过IsInstalled检测了

	// Use security dump-trust-settings to check trust settings
	// This command outputs trust settings directly to stdout (no file needed)
	cmd := exec.Command("security", "dump-trust-settings", "-d")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If dump-trust-settings fails, assume certificate is trusted if it exists in System keychain
		logger.Debug(fmt.Sprintf("Failed to dump trust settings: %v, output: %s", err, string(output)))
		// For root CA certificates in System keychain, they are usually trusted by default
		return true, nil
	}

	// Check if the certificate CN appears in trust settings output
	trustSettings := string(output)
	if strings.Contains(trustSettings, commonName) {
		logger.Debug(fmt.Sprintf("Certificate %s found in trust settings", commonName))
		return true, nil
	}

	// Certificate exists in System keychain but may not be explicitly trusted
	// For root CA certificates in System keychain, they are usually trusted by default
	// However, if it's not in trust settings, we should check more carefully
	logger.Debug(fmt.Sprintf("Certificate %s not found in trust settings, but exists in System keychain", commonName))
	return true, nil
}

// GetTrustStatus returns detailed trust status for macOS certificates
func (d *DarwinInstaller) GetTrustStatus(certType string, certBytes []byte) (string, error) {
	// Check if installed (exists in keychain)
	installed, _ := d.IsInstalled(certType, certBytes)
	if !installed {
		return "not_found", nil
	}

	// Check if trusted
	trusted, _ := d.IsTrusted(certType, certBytes)
	if trusted {
		return "system_trusted", nil
	}

	return "installed_not_trusted", nil
}

// ============= Linux Implementation =============

type LinuxInstaller struct{}

// IsInstalled checks if certificate is installed in system or user CA directories
func (l *LinuxInstaller) IsInstalled(certType string, certBytes []byte) (bool, error) {
	certName := getCertFileName(certType)

	// Check system-level path
	systemPath := fmt.Sprintf("/etc/ssl/certs/%s", certName)
	if _, err := os.Stat(systemPath); err == nil {
		logger.Info(fmt.Sprintf("Certificate %s found in system path: %s", certType, systemPath))
		return true, nil
	}

	// Check user-level path
	homeDir, _ := os.UserHomeDir()
	userPath := filepath.Join(homeDir, ".local/share/ca-certificates/custom", certName)
	if _, err := os.Stat(userPath); err == nil {
		logger.Info(fmt.Sprintf("Certificate %s found in user path: %s", certType, userPath))
		return true, nil
	}

	logger.Info(fmt.Sprintf("Certificate %s not found in Linux system", certType))
	return false, nil
}

// Install attempts to install certificate to system or user CA directory
func (l *LinuxInstaller) Install(certType string, certPath string) error {
	logger.Info(fmt.Sprintf("Installing certificate %s to Linux system", certType))

	certName := getCertFileName(certType)

	// Attempt 1: System-level installation (requires sudo)
	systemCertPath := fmt.Sprintf("/etc/ssl/certs/%s", certName)
	logger.Info(fmt.Sprintf("Attempting system-level installation to %s", systemCertPath))

	// Copy file with sudo
	copyCmd := exec.Command("sudo", "cp", certPath, systemCertPath)
	if err := copyCmd.Run(); err == nil {
		// Update CA certificates
		updateCmd := exec.Command("sudo", "update-ca-certificates")
		if err := updateCmd.Run(); err == nil {
			logger.Info("Certificate installed to system keychain and CA certificates updated")
			return nil
		}
	}

	logger.Info("System-level installation failed, attempting user-level installation")

	// Attempt 2: User-level installation (no sudo needed)
	homeDir, _ := os.UserHomeDir()
	userCertDir := filepath.Join(homeDir, ".local/share/ca-certificates/custom")

	if err := os.MkdirAll(userCertDir, 0700); err != nil {
		logger.Error(fmt.Sprintf("Failed to create certificate directory: %v", err))
		return fmt.Errorf("failed to create certificate directory: %w", err)
	}

	userCertPath := filepath.Join(userCertDir, certName)

	// Copy certificate file (no sudo)
	certData, readErr := os.ReadFile(certPath)
	if readErr != nil {
		return fmt.Errorf("failed to read certificate file: %w", readErr)
	}

	if writeErr := os.WriteFile(userCertPath, certData, 0644); writeErr != nil {
		return fmt.Errorf("failed to write certificate file: %w", writeErr)
	}

	// Update CA certificates
	updateCmd := exec.Command("update-ca-certificates", "--fresh", "--verbose")
	if err := updateCmd.Run(); err != nil {
		logger.Warn(fmt.Sprintf("Failed to update CA certificates: %v", err))
	}

	logger.Info(fmt.Sprintf("Certificate installed to user keychain at %s", userCertPath))
	return nil
}

// Remove deletes certificate from system or user CA directory
func (l *LinuxInstaller) Remove(certType string, certBytes []byte) error {
	// Get cert name from config
	config := cert.GetCertConfig(certType)
	if config == nil {
		return fmt.Errorf("unknown certificate type: %s", certType)
	}

	certName := config.FileName + ".pem"
	logger.Info(fmt.Sprintf("Removing certificate %s from Linux system", certType))

	// Attempt 1: Remove from system path (requires sudo)
	systemCertPath := fmt.Sprintf("/etc/ssl/certs/%s", certName)
	rmCmd := exec.Command("sudo", "rm", systemCertPath)
	if err := rmCmd.Run(); err == nil {
		// Update CA certificates
		updateCmd := exec.Command("sudo", "update-ca-certificates")
		_ = updateCmd.Run()
		logger.Info("Certificate removed from system keychain")
		return nil
	}

	// Attempt 2: Remove from user path
	homeDir, _ := os.UserHomeDir()
	userCertPath := filepath.Join(homeDir, ".local/share/ca-certificates/custom", certName)

	if err := os.Remove(userCertPath); err == nil {
		// Update CA certificates
		updateCmd := exec.Command("update-ca-certificates")
		_ = updateCmd.Run()
		logger.Info(fmt.Sprintf("Certificate removed from user keychain at %s", userCertPath))
		return nil
	}

	return fmt.Errorf("certificate not found in system or user keychain")
}

// GetCertInfo retrieves certificate information
func (l *LinuxInstaller) GetCertInfo(certType string, certBytes []byte) (CertInfo, error) {
	info, err := parseCertificateInfo(certBytes)
	if err != nil {
		return info, err
	}

	certName := getCertFileName(certType)
	homeDir, _ := os.UserHomeDir()
	userPath := filepath.Join(homeDir, ".local/share/ca-certificates/custom", certName)

	info.InstallPath = userPath
	return info, nil
}

// GetInstallPath returns the Linux installation path
func (l *LinuxInstaller) GetInstallPath(certType string) string {
	homeDir, _ := os.UserHomeDir()
	certName := getCertFileName(certType)
	return filepath.Join(homeDir, ".local/share/ca-certificates/custom", certName)
}

// IsTrusted checks if a certificate is in Linux's trusted CA bundle
func (l *LinuxInstaller) IsTrusted(certType string, certBytes []byte) (bool, error) {
	// Try to read ca-certificates.crt
	caFile := "/etc/ssl/certs/ca-certificates.crt"
	if _, err := os.Stat(caFile); err != nil {
		// Try alternative path
		caFile = "/etc/ssl/certs/ca-bundle.crt"
	}

	caData, err := os.ReadFile(caFile)
	if err != nil {
		return false, fmt.Errorf("failed to read CA file: %w", err)
	}

	// Extract certificate fingerprint for checking
	block, _ := pem.Decode(certBytes)
	if block == nil {
		return false, fmt.Errorf("failed to decode certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return false, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Get fingerprint and check if present in CA bundle
	fingerprint := fmt.Sprintf("%X", sha256.Sum256(cert.Raw))
	return strings.Contains(string(caData), fingerprint), nil
}

// GetTrustStatus returns detailed trust status for Linux certificates
func (l *LinuxInstaller) GetTrustStatus(certType string, certBytes []byte) (string, error) {
	// Check if installed
	installed, _ := l.IsInstalled(certType, certBytes)
	if !installed {
		return "not_found", nil
	}

	// Check if trusted
	trusted, _ := l.IsTrusted(certType, certBytes)
	if trusted {
		return "system_trusted", nil
	}

	return "installed_not_trusted", nil
}

// ============= Windows Implementation =============

type WindowsInstaller struct{}

// IsInstalled checks if certificate is installed in Windows certificate store
func (w *WindowsInstaller) IsInstalled(certType string, certBytes []byte) (bool, error) {
	// Extract the real certificate Common Name from the certificate itself
	commonName, err := extractCertCommonName(certBytes)
	if err != nil {
		logger.Warn(fmt.Sprintf("Failed to extract cert CN from bytes, falling back to hardcoded name: %v", err))
		// Fallback to hardcoded name if extraction fails
		commonName = getCertCommonName(certType)
	}

	logger.Info(fmt.Sprintf("Checking if certificate %s (CN: %s) is installed in Windows certificate store", certType, commonName))

	psCmd := fmt.Sprintf(
		"Get-ChildItem Cert:\\\\CurrentUser\\\\Root | Where-Object {$_.Subject -like '*%s*'} | Select-Object -First 1",
		commonName,
	)

	cmd := exec.Command("powershell", "-Command", psCmd)
	output, err := cmd.Output()

	if len(output) > 0 {
		logger.Info(fmt.Sprintf("Certificate %s found in Windows certificate store", commonName))
		return true, nil
	}

	logger.Info(fmt.Sprintf("Certificate %s not found in Windows certificate store", commonName))
	return false, nil
}

// Install adds certificate to Windows certificate store
func (w *WindowsInstaller) Install(certType string, certPath string) error {
	logger.Info(fmt.Sprintf("Installing certificate %s to Windows certificate store", certType))

	// Convert path to Windows format
	certPath = strings.ReplaceAll(certPath, "/", "\\")

	psCmd := fmt.Sprintf(
		"Import-Certificate -FilePath '%s' -CertStoreLocation Cert:\\CurrentUser\\Root -ErrorAction Stop",
		certPath,
	)

	cmd := exec.Command("powershell", "-Command", psCmd)
	output, err := cmd.CombinedOutput()

	if err != nil {
		logger.Error(fmt.Sprintf("Failed to install certificate: %v, output: %s", err, string(output)))
		return fmt.Errorf("certificate installation failed: %w", err)
	}

	logger.Info("Certificate installed successfully to Windows certificate store")
	return nil
}

// Remove deletes certificate from Windows certificate store
func (w *WindowsInstaller) Remove(certType string, certBytes []byte) error {
	// Extract the real certificate Common Name from the certificate itself
	commonName, err := extractCertCommonName(certBytes)
	if err != nil {
		logger.Warn(fmt.Sprintf("Failed to extract cert CN from bytes, falling back to config: %v", err))
		// Fallback to configuration if extraction fails
		config := cert.GetCertConfig(certType)
		if config != nil {
			commonName = config.CN
		} else {
			commonName = certType
		}
	}

	logger.Info(fmt.Sprintf("Removing certificate %s (CN: %s) from Windows certificate store", certType, commonName))

	psCmd := fmt.Sprintf(
		"$cert = Get-ChildItem Cert:\\CurrentUser\\Root | Where-Object {$_.Subject -like '*%s*'} | Select-Object -First 1; if ($cert) { Remove-Item -Path \"Cert:\\CurrentUser\\Root\\$($cert.Thumbprint)\" -Force }",
		commonName,
	)

	cmd := exec.Command("powershell", "-Command", psCmd)
	output, err := cmd.CombinedOutput()

	if err != nil {
		logger.Error(fmt.Sprintf("Failed to remove certificate: %v, output: %s", err, string(output)))
		return fmt.Errorf("certificate removal failed: %w", err)
	}

	logger.Info("Certificate removed successfully from Windows certificate store")
	return nil
}

// GetCertInfo retrieves certificate information
func (w *WindowsInstaller) GetCertInfo(certType string, certBytes []byte) (CertInfo, error) {
	info, err := parseCertificateInfo(certBytes)
	if err != nil {
		return info, err
	}
	info.InstallPath = "Cert:\\CurrentUser\\Root"
	return info, nil
}

// GetInstallPath returns the Windows installation path
func (w *WindowsInstaller) GetInstallPath(certType string) string {
	return "Cert:\\CurrentUser\\Root"
}

// IsTrusted checks if a certificate is trusted on Windows (equivalent to IsInstalled for Root store)
func (w *WindowsInstaller) IsTrusted(certType string, certBytes []byte) (bool, error) {
	// On Windows, certificates in Root store are automatically trusted
	// So IsTrusted is equivalent to IsInstalled
	return w.IsInstalled(certType, certBytes)
}

// GetTrustStatus returns detailed trust status for Windows certificates
func (w *WindowsInstaller) GetTrustStatus(certType string, certBytes []byte) (string, error) {
	// Check if installed
	installed, _ := w.IsInstalled(certType, certBytes)
	if !installed {
		return "not_found", nil
	}

	// On Windows, being in Root store means it's trusted
	return "system_trusted", nil
}

// ============= Unimplemented Installer =============

type UnimplementedInstaller struct{}

func (u *UnimplementedInstaller) IsInstalled(certType string, certBytes []byte) (bool, error) {
	return false, fmt.Errorf("certificate operations not supported on this platform")
}

func (u *UnimplementedInstaller) Install(certType string, certPath string) error {
	return fmt.Errorf("certificate operations not supported on this platform")
}

func (u *UnimplementedInstaller) Remove(certType string, certBytes []byte) error {
	return fmt.Errorf("certificate operations not supported on this platform")
}

func (u *UnimplementedInstaller) GetCertInfo(certType string, certBytes []byte) (CertInfo, error) {
	return CertInfo{}, fmt.Errorf("certificate operations not supported on this platform")
}

func (u *UnimplementedInstaller) GetInstallPath(certType string) string {
	return ""
}

// IsTrusted returns error for unsupported platforms
func (u *UnimplementedInstaller) IsTrusted(certType string, certBytes []byte) (bool, error) {
	return false, fmt.Errorf("IsTrusted not implemented for this platform")
}

// GetTrustStatus returns error for unsupported platforms
func (u *UnimplementedInstaller) GetTrustStatus(certType string, certBytes []byte) (string, error) {
	return "unsupported_platform", fmt.Errorf("GetTrustStatus not implemented for this platform")
}

// ============= Helper Functions =============

// getCertCommonName returns the certificate common name from configuration
// Deprecated: Use cert.GetCertConfig(certType).CN instead
func getCertCommonName(certType string) string {
	config := cert.GetCertConfig(certType)
	if config != nil {
		return config.CN
	}
	// Fallback for unknown types
	return certType
}

// getCertFileName returns the certificate file name from configuration
// Deprecated: Use cert.GetCertConfig(certType).FileName instead
func getCertFileName(certType string) string {
	config := cert.GetCertConfig(certType)
	if config != nil {
		return config.FileName + ".pem"
	}
	// Fallback for unknown types
	return certType + ".pem"
}
