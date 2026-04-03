package runtimepath

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type Mode string

const (
	ModeInteractive Mode = "interactive"
	ModeDaemon      Mode = "daemon"

	UserStateDirName = ".aliang"
	CacheDirEnvVar   = "ALIANG_CACHE_DIR"
)

// DetectMode infers whether the current process is running as an interactive
// app/CLI or as a background daemon/service.
func DetectMode() Mode {
	if strings.TrimSpace(os.Getenv("ALIANG_DATA_DIR")) != "" {
		return ModeDaemon
	}
	if strings.TrimSpace(os.Getenv("ALIANG_SOCKET_PATH")) != "" {
		return ModeDaemon
	}
	return ModeInteractive
}

func BinaryFilename() string {
	if runtime.GOOS == "windows" {
		return "aliang.exe"
	}
	return "aliang"
}

func CoreDataDir() string {
	if dir := strings.TrimSpace(os.Getenv("ALIANG_DATA_DIR")); dir != "" {
		return filepath.Clean(dir)
	}
	switch runtime.GOOS {
	case "darwin":
		return "/Library/Application Support/one.aliang.aliang"
	case "linux":
		return "/var/lib/aliang"
	case "windows":
		return os.ExpandEnv(`${ProgramData}\Aliang`)
	default:
		return "/var/lib/aliang"
	}
}

func CoreLogDir() string {
	if dir := strings.TrimSpace(os.Getenv("ALIANG_LOG_DIR")); dir != "" {
		return filepath.Clean(dir)
	}
	switch runtime.GOOS {
	case "darwin":
		return "/Library/Logs/Aliang"
	case "linux":
		return "/var/log/aliang"
	case "windows":
		return os.ExpandEnv(`${ProgramData}\Aliang\logs`)
	default:
		return "/var/log/aliang"
	}
}

func CoreSocketPath() string {
	if path := strings.TrimSpace(os.Getenv("ALIANG_SOCKET_PATH")); path != "" {
		if runtime.GOOS == "windows" && !strings.HasPrefix(path, `\\.\pipe\`) {
			return `\\.\pipe\aliang-core`
		}
		return path
	}
	switch runtime.GOOS {
	case "darwin":
		return "/var/run/aliang-core.sock"
	case "linux":
		return "/run/aliang-core.sock"
	case "windows":
		return `\\.\pipe\aliang-core`
	default:
		return "/tmp/aliang-core.sock"
	}
}

func UserHomeDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		if home := strings.TrimSpace(os.Getenv("USERPROFILE")); home != "" {
			return home, nil
		}
	default:
		if home := strings.TrimSpace(os.Getenv("HOME")); home != "" {
			return home, nil
		}
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	homeDir = strings.TrimSpace(homeDir)
	if homeDir == "" {
		return "", errors.New("user home directory is empty")
	}
	return homeDir, nil
}

func UserStateDir() (string, error) {
	homeDir, err := UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, UserStateDirName), nil
}

func UserConfigPath() (string, error) {
	stateDir, err := UserStateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(stateDir, "config.json"), nil
}

func RuntimeConfigPath() string {
	return filepath.Join(CoreDataDir(), "config.json")
}

func RuntimeExecutablePath() string {
	return filepath.Join(CoreDataDir(), BinaryFilename())
}

func ExpandHome(path string) (string, error) {
	if path == "" || path[0] != '~' {
		return path, nil
	}

	homeDir, err := UserHomeDir()
	if err != nil {
		return "", err
	}

	if len(path) == 1 {
		return homeDir, nil
	}
	return filepath.Join(homeDir, path[1:]), nil
}

// ResolveStateDir returns the canonical directory for local mutable state such
// as sqlite files, logs, generated certificates, and GeoIP databases.
func ResolveStateDir() (string, error) {
	if envDir := strings.TrimSpace(os.Getenv(CacheDirEnvVar)); envDir != "" {
		expandedDir, err := ExpandHome(envDir)
		if err != nil {
			return "", err
		}
		return filepath.Clean(expandedDir), nil
	}

	if DetectMode() == ModeDaemon {
		return filepath.Clean(CoreDataDir()), nil
	}

	userStateDir, err := UserStateDir()
	if err != nil {
		return "", err
	}
	return filepath.Clean(userStateDir), nil
}

func ResolveDefaultConfigPathForMode(mode Mode, logicalPath string) (string, error) {
	logicalPath = strings.TrimSpace(logicalPath)
	if logicalPath == "" {
		return "", errors.New("config path is empty")
	}

	if mode == ModeDaemon {
		return filepath.Join(CoreDataDir(), filepath.Base(logicalPath)), nil
	}

	if strings.HasPrefix(logicalPath, "~") {
		path, err := UserConfigPath()
		if err != nil {
			return "", err
		}
		return filepath.Clean(path), nil
	}

	return filepath.Clean(logicalPath), nil
}
