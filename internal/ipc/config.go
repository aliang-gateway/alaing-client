package ipc

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Socket path based on platform
func defaultSocketPath() string {
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

// SocketPath returns the IPC socket path.
// Uses ALIANG_SOCKET_PATH env var if set, otherwise defaults.
func SocketPath() string {
	if path := os.Getenv("ALIANG_SOCKET_PATH"); path != "" {
		return path
	}
	return defaultSocketPath()
}

// CoreDataDir returns the system-level data directory.
// Uses ALIANG_DATA_DIR env var if set, otherwise defaults.
func CoreDataDir() string {
	if dir := os.Getenv("ALIANG_DATA_DIR"); dir != "" {
		return dir
	}
	switch runtime.GOOS {
	case "darwin":
		return "/Library/Application Support/org.nursor.aliang"
	case "linux":
		return "/var/lib/aliang"
	case "windows":
		return os.ExpandEnv(`${ProgramData}\Aliang`)
	default:
		return "/var/lib/aliang"
	}
}

// CoreLogDir returns the system-level log directory.
func CoreLogDir() string {
	if dir := os.Getenv("ALIANG_LOG_DIR"); dir != "" {
		return dir
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

// EnsureCoreDirs ensures all required directories exist.
func EnsureCoreDirs() error {
	dirs := []string{
		CoreDataDir(),
		filepath.Dir(SocketPath()),
		CoreLogDir(),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}
