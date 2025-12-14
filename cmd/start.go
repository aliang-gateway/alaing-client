package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	httpServer "nursor.org/nursorgate/app/http"
	"nursor.org/nursorgate/common/logger"
	auth "nursor.org/nursorgate/processor/auth"
	"nursor.org/nursorgate/processor/runtime"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the nonelane server",
	Long: `Start the nonelane server with configuration from file or default embedded config.

Examples:
  # Start with local config file and activate user with token
  nonelane start --config ./config.json --token your-token-here

  # Start with local config file (use locally saved user info)
  nonelane start --config ./config.json

  # Start with default embedded configuration and activate user with token
  nonelane start --token your-token-here

  # Start with default embedded configuration (use locally saved user info)
  nonelane start`,
	RunE: runStart,
}

func init() {
	rootCmd.AddCommand(startCmd)
	// Parameters are inherited from root command via PersistentFlags
}

func ApplyDefaultConfig() error {
	logger.Info("Loading default embedded configuration...")
	defaultConfigBytes := GetDefaultConfigBytes()
	config, err := LoadConfigFromBytes(defaultConfigBytes)
	if err != nil {
		return fmt.Errorf("failed to load default config: %w", err)
	}

	if err := ApplyConfig(config); err != nil {
		return fmt.Errorf("failed to apply default config: %w", err)
	}

	// Mark that we're using the default configuration
	setUseDefaultConfig(true)

	logger.Info("Default configuration applied successfully")
	return nil
}

func runStart(cmd *cobra.Command, args []string) error {
	// Get the global startup state
	startupState := runtime.GetStartupState()

	// Load configuration (from file or default)
	if configPath != "" {
		// Load from local config file
		logger.Info(fmt.Sprintf("Loading configuration from file: %s", configPath))
		if err := LoadAndApplyConfig(configPath); err != nil {
			return fmt.Errorf("failed to load config from file: %w", err)
		}
	} else {
		// Use default embedded configuration
		logger.Info("No config file provided, using embedded default configuration...")
		if err := ApplyDefaultConfig(); err != nil {
			return fmt.Errorf("failed to apply default config: %w", err)
		}
		setUseDefaultConfig(true)
	}

	// Determine initial startup status based on configuration and available resources
	// Status reflects whether the system is ready for proxy operations
	initialStatus := determineInitialStartupStatus(token)
	startupState.SetStatus(initialStatus)
	logger.Info(fmt.Sprintf("Initial startup status: %s", initialStatus))

	// Initialize user info (activate with token or load locally saved info)
	// This will transition the startup state based on success/failure
	// and will fetch proxyserver configuration if applicable
	// Note: Returns error only if token activation fails, causing startup to fail
	if err := InitializeUser(token); err != nil {
		return err
	}

	// 启动服务器
	logger.Info("Starting nonelane server...")

	// 启动 HTTP 服务器（包含代理注册中心初始化）
	// HTTP server always starts, but API requests are gated by startup status middleware
	go httpServer.StartHttpServer()

	// 等待信号并优雅关闭
	logger.Info("Server started successfully. Press Ctrl+C to stop.")

	// 设置信号处理器
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 等待信号
	<-sigChan
	logger.Info("Received shutdown signal, stopping server...")

	// 优雅关闭：停止Token定时刷新
	auth.StopTokenRefresh()

	logger.Info("Server stopped successfully.")
	return nil
}

// determineInitialStartupStatus determines the initial startup status based on:
// 1. Whether a token is provided via --token flag
// 2. Whether local user information exists
// 3. Whether using default configuration
//
// The startup status indicates system readiness:
// - UNCONFIGURED: No token, no local user info → system awaiting configuration
// - CONFIGURING: Token provided → system attempting activation and fetch
// - READY: Has local user info AND using default config → system ready (if fetch succeeds)
// - CONFIGURED: Has local user info BUT NOT using default config → needs fetch
func determineInitialStartupStatus(tokenFlag string) runtime.StartupStatus {
	// Check if local user info exists by trying to get its path
	hasLocalUserInfo := false
	if userInfoPath, err := auth.GetUserInfoPath(); err == nil {
		if _, err := os.Stat(userInfoPath); err == nil {
			hasLocalUserInfo = true
		}
	}

	// Determine status based on conditions
	if tokenFlag != "" {
		// Token provided → system is in configuration process
		logger.Debug("Token provided via --token, status: CONFIGURING")
		return runtime.CONFIGURING
	}

	if hasLocalUserInfo && IsUsingDefaultConfig() {
		// Local user info exists AND using default config → likely ready
		// (actual READY status depends on fetch success in InitializeUser)
		logger.Debug("Local user info found and using default config, initial status: CONFIGURED")
		return runtime.CONFIGURED
	}

	if hasLocalUserInfo {
		// Local user info exists but custom config → needs to determine readiness via fetch
		logger.Debug("Local user info found but using custom config, initial status: CONFIGURED")
		return runtime.CONFIGURED
	}

	// No token, no local user info → unconfigured
	logger.Info("No token and no local user info found, status: UNCONFIGURED")
	return runtime.UNCONFIGURED
}

// ===== Test-Only Exports =====
// These functions are exported for testing purposes only

// ResetGlobalStartupStateForTest resets the startup state singleton for testing
// This allows tests to run in isolation without state pollution
func ResetGlobalStartupStateForTest() {
	runtime.ResetGlobalStartupStateForTest()
}

// DetermineInitialStartupStatusForTest exports the internal function for testing
func DetermineInitialStartupStatusForTest(tokenFlag string) runtime.StartupStatus {
	return determineInitialStartupStatus(tokenFlag)
}
