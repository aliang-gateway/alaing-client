package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	httpServer "nursor.org/nursorgate/app/http"
	"nursor.org/nursorgate/common/cache"
	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/common/model"
	auth "nursor.org/nursorgate/processor/auth"
	"nursor.org/nursorgate/processor/config"
	"nursor.org/nursorgate/processor/geoip"
	"nursor.org/nursorgate/processor/nacos"
	"nursor.org/nursorgate/processor/rules"
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

	// ✅ GLOBAL Rule Engine Initialization (Phase 5: US5)
	// This should be done once at startup, NOT separately in HTTP and TUN modes
	// InitializeGlobalRuleEngine handles:
	// 1. Rule engine initialization (singleton)
	// 2. GeoIP database loading (if enabled in config)
	// 3. Nacos configuration preloading
	if err := InitializeGlobalRuleEngine(); err != nil {
		logger.Error(fmt.Sprintf("Failed to initialize global rule engine: %v", err))
		// Don't fail startup - system can run with default configuration
	}

	// T061: Initialize Nacos configuration manager (T055-T063)
	// This loads routing configuration and starts listener if auto_update=true
	var nacosManager *nacos.ConfigManager
	cfg := config.GetGlobalConfig()
	if cfg != nil && cfg.NacosServer != "" {
		logger.Info("Initializing Nacos configuration manager...")
		manager, err := nacos.InitializeFromConfig(cfg.NacosServer)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to initialize Nacos: %v", err))
			// Don't fail startup if Nacos initialization fails
			// System can still run with default configuration
		} else {
			nacosManager = manager
			logger.Info("Nacos configuration manager initialized successfully")
		}
	} else {
		logger.Info("Nacos server not configured, skipping Nacos initialization")
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

	// T062: Graceful shutdown - stop Nacos listener if initialized
	if nacosManager != nil {
		nacos.GracefulShutdown(nacosManager)
	}

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

// InitializeGlobalRuleEngine initializes the global rule engine once at startup
// This is the ONLY place where rule engine should be initialized
// Replaces duplicate initialization in:
// - app/http/server.go:initializeRuleEngine()
// - inbound/tun/runner/start.go:initializeRuleEngineForTUN()
func InitializeGlobalRuleEngine() error {
	logger.Info("========================================")
	logger.Info("Global Rule Engine Initialization")
	logger.Info("========================================")

	// Step 1: Create default routing rules configuration
	logger.Info("Step 1: Creating default routing rules configuration...")
	defaultRules := model.NewRoutingRulesConfig()

	// Step 2: Initialize Rule Engine (singleton)
	logger.Info("Step 2: Initializing rule engine...")
	ruleEngine := rules.GetEngine()
	if err := ruleEngine.Initialize(defaultRules); err != nil {
		return fmt.Errorf("failed to initialize rule engine: %w", err)
	}
	logger.Info("✓ Rule engine initialized")

	// Step 3: Load GeoIP database if enabled
	logger.Info("Step 3: Loading GeoIP database...")
	if defaultRules.Settings.GeoIPEnabled {
		if err := initializeGeoIPDatabase(); err != nil {
			logger.Warn(fmt.Sprintf("Failed to load GeoIP database (non-fatal): %v", err))
			logger.Warn("GeoIP routing will be disabled")
			// Disable GeoIP in geoip service
			geoipService := geoip.GetService()
			geoipService.Disable()
		} else {
			logger.Info("✓ GeoIP database loaded successfully")
		}
	} else {
		logger.Info("GeoIP routing is disabled in configuration (Settings.GeoIPEnabled=false)")
	}

	// Step 4: Preload Nacos configuration
	logger.Info("Step 4: Preloading Nacos configuration...")
	startTime := time.Now()
	_ = model.NewAllowProxyDomain()
	duration := time.Since(startTime)
	logger.Info(fmt.Sprintf("✓ Nacos configuration loaded in %v", duration))

	logger.Info("========================================")
	logger.Info("✅ Global Rule Engine Initialization Complete")
	logger.Info("========================================")

	return nil
}

// initializeGeoIPDatabase loads the GeoIP database from default location
// Default path: ~/.nonelane/GeoLite2-Country.mmdb
func initializeGeoIPDatabase() error {
	// Get home directory
	homeDir, err := cache.ExpandHomePath("~")
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// GeoIP database path: ~/.nonelane/GeoLite2-Country.mmdb
	geoipPath := filepath.Join(homeDir, ".nonelane", "GeoLite2-Country.mmdb")

	// Load database
	logger.Info(fmt.Sprintf("Loading GeoIP database from: %s", geoipPath))
	geoipService := geoip.GetService()
	if err := geoipService.LoadDatabase(geoipPath); err != nil {
		return fmt.Errorf("failed to load GeoIP database: %w", err)
	}

	logger.Info(fmt.Sprintf("✓ GeoIP database loaded from %s", geoipPath))
	return nil
}
