package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	httpServer "nursor.org/nursorgate/app/http"
	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/inbound/http"
	"nursor.org/nursorgate/inbound/tun/runner"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the nursor server",
	Long: `Start the nursor server with configuration from file or remote server.

Examples:
  # Start with local config file
  nursor start --config ./config.json

  # Start with token (fetch config from remote)
  nursor start --token your-token-here

  # Start with token and custom server URL
  nursor start --token your-token-here --server https://api.example.com

  # Start with default embedded configuration
  nursor start`,
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
	// 检查参数
	if configPath == "" && token == "" {
		// Use default config
		logger.Info("No configuration provided, using embedded default configuration...")
		if err := ApplyDefaultConfig(); err != nil {
			return fmt.Errorf("failed to apply default config: %w", err)
		}
		setUseDefaultConfig(true)
	} else if configPath != "" {
		// 方式1: 从本地文件加载配置
		logger.Info(fmt.Sprintf("Loading configuration from file: %s", configPath))
		if err := LoadAndApplyConfig(configPath); err != nil {
			return fmt.Errorf("failed to load config from file: %w", err)
		}
	} else if token != "" {
		// 方式2: 从远程服务器获取配置
		logger.Info("Fetching configuration from remote server...")
		if err := FetchAndApplyConfigFromRemote(token, serverURL); err != nil {
			return fmt.Errorf("failed to fetch config from remote: %w", err)
		}
	}

	// 启动服务器
	logger.Info("Starting nursor server...")

	// 启动 HTTP 服务器（包含代理注册中心初始化）
	go httpServer.StartHttpServer()

	// 启动 TUN 服务
	if startTun {
		go func() {
			runner.Start()
		}()
	} else if startHttp {
		go func() {
			http.StartMitmHttp()
		}()
	}

	// 等待信号
	logger.Info("Server started successfully. Press Ctrl+C to stop.")

	// 使用信号处理来优雅关闭
	select {} // 阻塞主线程，实际应该使用 signal.Notify
}
