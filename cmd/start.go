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

	logger.Info("Default configuration applied successfully")
	return nil
}

func runStart(cmd *cobra.Command, args []string) error {
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

	// Initialize user info (activate with token or load locally saved info)
	// If --token is provided, activate user; otherwise load locally saved user info
	InitializeUser(token)

	// 启动服务器
	logger.Info("Starting nonelane server...")

	// 启动 HTTP 服务器（包含代理注册中心初始化）
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
