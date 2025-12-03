package logger

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// LogLevel represents logging level
type LogLevelType int

const (
	TRACE LogLevelType = iota
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
	PANIC
)

// LogConfig consolidates all logger configuration
type LogConfig struct {
	// Log level for main logger
	Level LogLevelType
	// Error deduplication time window
	ErrorWindow time.Duration
	// Maximum error count within time window
	MaxErrorCount int
	// Cleanup interval for expired errors
	CleanupInterval time.Duration
	// File logging path (main logger)
	FileLogPath string
	// Enable file rotation
	EnableFileRotation bool
	// Max log file size in bytes
	MaxLogSize int64
	// Max number of backups
	MaxLogBackups int
	// Sentry DSN
	SentryDSN string
	// Enable Sentry integration
	EnableSentry bool
}

// DefaultLogConfig returns default configuration
func DefaultLogConfig() *LogConfig {
	homeDir := getHomeDir()
	logDir := filepath.Join(homeDir, ".nursor")

	return &LogConfig{
		Level:              WARN,
		ErrorWindow:        1 * time.Hour,
		MaxErrorCount:      4,
		CleanupInterval:    2 * time.Hour,
		FileLogPath:        filepath.Join(logDir, "nursor_core.log"),
		EnableFileRotation: true,
		MaxLogSize:         100 * 1024 * 1024, // 100MB
		MaxLogBackups:      5,
		SentryDSN:          os.Getenv("SENTRY_DSN"),
		EnableSentry:       os.Getenv("SENTRY_DSN") != "",
	}
}

// HTTPLogConfig returns HTTP logger configuration
func HTTPLogConfig() *LogConfig {
	homeDir := getHomeDir()
	var logPath string
	if runtime.GOOS == "darwin" {
		logPath = filepath.Join("/Library/Logs/Nursor", "nursor_http.log")
	} else {
		logPath = filepath.Join(homeDir, ".nursor", "nursor_http.log")
	}

	return &LogConfig{
		Level:              INFO,
		FileLogPath:        logPath,
		EnableFileRotation: true,
		MaxLogSize:         50 * 1024 * 1024, // 50MB
		MaxLogBackups:      3,
	}
}

func getHomeDir() string {
	if runtime.GOOS == "darwin" {
		return "/Library/Logs/Nursor"
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp"
	}
	return home
}

// Global configuration instance
var (
	globalLogConfig   = DefaultLogConfig()
	globalLogConfigMu sync.RWMutex
)

// GetLogConfig returns current configuration
func GetLogConfig() *LogConfig {
	globalLogConfigMu.RLock()
	defer globalLogConfigMu.RUnlock()
	return globalLogConfig
}

// SetLogConfig updates the global configuration
func SetLogConfig(config *LogConfig) {
	globalLogConfigMu.Lock()
	defer globalLogConfigMu.Unlock()
	if config != nil {
		globalLogConfig = config
	}
}

// UpdateLogLevel updates the log level dynamically
func UpdateLogLevel(level LogLevelType) {
	globalLogConfigMu.Lock()
	defer globalLogConfigMu.Unlock()
	globalLogConfig.Level = level
}

// Legacy compatibility - keep ErrorDedupConfig for backward compatibility
type ErrorDedupConfig struct {
	ErrorWindow     time.Duration
	MaxErrorCount   int
	CleanupInterval time.Duration
}

// SetErrorDedupConfig updates configuration (legacy function)
func SetErrorDedupConfig(config *ErrorDedupConfig) {
	if config != nil {
		globalLogConfigMu.Lock()
		defer globalLogConfigMu.Unlock()
		globalLogConfig.ErrorWindow = config.ErrorWindow
		globalLogConfig.MaxErrorCount = config.MaxErrorCount
		globalLogConfig.CleanupInterval = config.CleanupInterval
	}
}
