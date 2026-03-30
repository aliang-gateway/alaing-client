//go:build windows

package services

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"nursor.org/nursorgate/common/logger"
)

const (
	wintunArchiveName   = "wintun-0.14.1.zip"
	wintunExtractedName = "wintun-0.14.1"
)

type windowsWintunDependencyController struct {
	mu    sync.Mutex
	state WintunDependencyStatus
}

func newWintunDependencyController() wintunDependencyController {
	controller := &windowsWintunDependencyController{
		state: WintunDependencyStatus{
			Supported:    true,
			Required:     true,
			Available:    false,
			Installing:   false,
			State:        "checking",
			Message:      "Checking Windows Wintun dependency.",
			Architecture: runtime.GOARCH,
			DownloadURL:  wintunDownloadURL,
			UpdatedAt:    time.Now().Unix(),
		},
	}
	controller.refreshLocked()
	return controller
}

func (c *windowsWintunDependencyController) Status() WintunDependencyStatus {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.state.Installing {
		c.refreshLocked()
	}
	return c.state
}

func (c *windowsWintunDependencyController) Refresh() WintunDependencyStatus {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.state.Installing {
		c.refreshLocked()
	}
	return c.state
}

func (c *windowsWintunDependencyController) StartInstall() WintunDependencyStatus {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state.Installing {
		return c.state
	}

	c.refreshLocked()
	if c.state.Available {
		return c.state
	}

	c.state.Installing = true
	c.state.State = "queued"
	c.state.Message = "Preparing Wintun dependency installation in the background."
	c.state.Error = ""
	c.state.UpdatedAt = time.Now().Unix()

	go c.install()

	return c.state
}

func (c *windowsWintunDependencyController) install() {
	logger.Info("Starting background Wintun dependency installation")

	if err := c.installWintun(); err != nil {
		message := err.Error()
		if os.IsPermission(err) {
			message = "Installing Wintun requires administrator permissions to write into the Windows system directory."
		}
		logger.Error(fmt.Sprintf("Wintun dependency installation failed: %v", err))

		c.mu.Lock()
		c.state.Installing = false
		c.state.Available = false
		c.state.State = "failed"
		c.state.Message = "Wintun dependency installation failed."
		c.state.Error = message
		c.state.LastChecked = time.Now().Unix()
		c.state.UpdatedAt = time.Now().Unix()
		c.mu.Unlock()
		return
	}

	c.mu.Lock()
	c.refreshLocked()
	c.state.Installing = false
	if c.state.Available {
		c.state.State = "installed"
		c.state.Message = "Wintun dependency is installed and ready for TUN mode."
		c.state.Error = ""
	} else {
		c.state.State = "failed"
		c.state.Message = "Wintun installation finished, but the DLL was not found in the Windows system directory."
		c.state.Error = "Wintun installation could not be verified after completion."
	}
	c.state.UpdatedAt = time.Now().Unix()
	c.mu.Unlock()
}

func (c *windowsWintunDependencyController) installWintun() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to resolve user home directory: %w", err)
	}

	cacheDir := filepath.Join(homeDir, ".aliang")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return fmt.Errorf("failed to create wintun cache directory: %w", err)
	}

	archivePath := filepath.Join(cacheDir, wintunArchiveName)
	extractDir := filepath.Join(cacheDir, wintunExtractedName)
	defer func() {
		if err := os.Remove(archivePath); err != nil && !os.IsNotExist(err) {
			logger.Warn("Failed to remove temporary Wintun archive", "path", archivePath, "error", err)
		}
		if err := os.RemoveAll(extractDir); err != nil && !os.IsNotExist(err) {
			logger.Warn("Failed to remove extracted Wintun directory", "path", extractDir, "error", err)
		}
	}()

	targetDir, sourceSubdir, err := resolveWintunInstallTarget()
	if err != nil {
		return err
	}
	targetPath := filepath.Join(targetDir, "wintun.dll")
	preferredDownloadURL := resolvePreferredWintunDownloadURL(runtime.GOARCH)

	if installedPath, ok := detectInstalledWintun(); ok {
		c.updateProgress("installed", "Wintun dependency is already installed.", installedPath, targetPath, "")
		return nil
	}

	if preferredDownloadURL != "" {
		c.updateProgress("downloading", "Downloading Wintun dependency directly from the mirror.", "", targetPath, "")
		if err := downloadFile(targetPath, preferredDownloadURL); err == nil {
			return nil
		} else {
			logger.Warn("Direct Wintun mirror download failed, falling back to official package", "url", preferredDownloadURL, "error", err)
			c.updateProgress("downloading", "Mirror download failed. Falling back to the official Wintun package.", "", targetPath, "")
		}
	}

	c.updateProgress("downloading", "Downloading Wintun dependency package from the official source.", "", targetPath, "")
	if err := downloadFile(archivePath, wintunDownloadURL); err != nil {
		return fmt.Errorf("failed to download Wintun package: %w", err)
	}

	c.updateProgress("extracting", "Extracting Wintun dependency package.", "", targetPath, "")
	if err := extractZipArchive(archivePath, extractDir); err != nil {
		return fmt.Errorf("failed to extract Wintun package: %w", err)
	}

	sourcePath := filepath.Join(extractDir, "wintun", "bin", sourceSubdir, "wintun.dll")
	if _, err := os.Stat(sourcePath); err != nil {
		return fmt.Errorf("failed to locate extracted Wintun DLL for %s: %w", sourceSubdir, err)
	}

	c.updateProgress("installing", "Installing Wintun into the Windows system directory.", sourcePath, targetPath, "")
	if err := copyFile(sourcePath, targetPath); err != nil {
		return fmt.Errorf("failed to install Wintun DLL to %s: %w", targetPath, err)
	}

	return nil
}

func (c *windowsWintunDependencyController) refreshLocked() {
	targetDir, _, err := resolveWintunInstallTarget()
	targetPath := ""
	if err == nil {
		targetPath = filepath.Join(targetDir, "wintun.dll")
	}

	if installedPath, ok := detectInstalledWintun(); ok {
		c.state.Supported = true
		c.state.Required = true
		c.state.Available = true
		c.state.State = "installed"
		c.state.Message = "Wintun dependency is available for TUN mode."
		c.state.Error = ""
		c.state.InstallPath = installedPath
		c.state.TargetPath = targetPath
		c.state.Architecture = runtime.GOARCH
		c.state.DownloadURL = resolvePreferredWintunDownloadURL(runtime.GOARCH)
		c.state.LastChecked = time.Now().Unix()
		c.state.UpdatedAt = time.Now().Unix()
		return
	}

	c.state.Supported = true
	c.state.Required = true
	c.state.Available = false
	if !c.state.Installing {
		c.state.State = "missing"
		c.state.Message = "Wintun dependency is missing. Install it before enabling TUN mode."
		c.state.Error = ""
	}
	c.state.InstallPath = ""
	c.state.TargetPath = targetPath
	c.state.Architecture = runtime.GOARCH
	c.state.DownloadURL = resolvePreferredWintunDownloadURL(runtime.GOARCH)
	c.state.LastChecked = time.Now().Unix()
	c.state.UpdatedAt = time.Now().Unix()
}

func (c *windowsWintunDependencyController) updateProgress(state string, message string, installPath string, targetPath string, errMsg string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.Supported = true
	c.state.Required = true
	c.state.Installing = true
	c.state.Available = false
	c.state.State = state
	c.state.Message = message
	c.state.Error = errMsg
	c.state.InstallPath = installPath
	c.state.TargetPath = targetPath
	c.state.Architecture = runtime.GOARCH
	c.state.DownloadURL = resolvePreferredWintunDownloadURL(runtime.GOARCH)
	c.state.UpdatedAt = time.Now().Unix()
}

func downloadFile(destination string, url string) error {
	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status %d", resp.StatusCode)
	}

	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}

	tempPath := destination + ".tmp"
	file, err := os.Create(tempPath)
	if err != nil {
		return err
	}

	if _, err := io.Copy(file, resp.Body); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}

	if err := os.Remove(destination); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(tempPath, destination)
}

func extractZipArchive(archivePath string, destination string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	if err := os.RemoveAll(destination); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}

	for _, file := range reader.File {
		targetPath := filepath.Join(destination, file.Name)
		if !strings.HasPrefix(targetPath, filepath.Clean(destination)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid zip entry path: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}

		src, err := file.Open()
		if err != nil {
			return err
		}

		dst, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, file.Mode())
		if err != nil {
			src.Close()
			return err
		}

		_, copyErr := io.Copy(dst, src)
		closeErr := dst.Close()
		srcErr := src.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if srcErr != nil {
			return srcErr
		}
	}

	return nil
}

func copyFile(source string, destination string) error {
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()

	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}

	tempPath := destination + ".tmp"
	dst, err := os.OpenFile(tempPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
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

	if err := os.Remove(destination); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(tempPath, destination)
}

func detectInstalledWintun() (string, bool) {
	for _, candidate := range wintunSearchPaths() {
		path := filepath.Join(candidate, "wintun.dll")
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, true
		}
	}
	return "", false
}

func wintunSearchPaths() []string {
	systemRoot := strings.TrimSpace(os.Getenv("WINDIR"))
	if systemRoot == "" {
		systemRoot = strings.TrimSpace(os.Getenv("SystemRoot"))
	}
	if systemRoot == "" {
		systemRoot = `C:\Windows`
	}

	candidates := make([]string, 0, 2)
	for _, dirName := range []string{"System32", "SysWOW64"} {
		dir := filepath.Join(systemRoot, dirName)
		if _, err := os.Stat(dir); err == nil {
			candidates = append(candidates, dir)
		}
	}
	return candidates
}

func resolveWintunInstallTarget() (targetDir string, sourceSubdir string, err error) {
	searchPaths := wintunSearchPaths()
	if len(searchPaths) == 0 {
		return "", "", fmt.Errorf("failed to locate Windows system directory")
	}

	system32 := searchPaths[0]
	syswow64 := ""
	if len(searchPaths) > 1 {
		syswow64 = searchPaths[1]
	}

	switch runtime.GOARCH {
	case "amd64":
		return system32, "amd64", nil
	case "arm64":
		return system32, "arm64", nil
	case "arm":
		return system32, "arm", nil
	case "386":
		if syswow64 != "" {
			return syswow64, "x86", nil
		}
		return system32, "x86", nil
	default:
		return "", "", fmt.Errorf("unsupported Windows architecture for Wintun: %s", runtime.GOARCH)
	}
}

func resolvePreferredWintunDownloadURL(goarch string) string {
	switch goarch {
	case "amd64", "386":
		return wintunMirrorURLX86
	case "arm64":
		return wintunMirrorURLARM64
	default:
		return ""
	}
}
