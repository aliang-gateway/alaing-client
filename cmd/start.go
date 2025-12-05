package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	httpServer "nursor.org/nursorgate/app/http"
	"nursor.org/nursorgate/common/logger"
	"nursor.org/nursorgate/inbound/http"
	"nursor.org/nursorgate/inbound/tun/runner"
)

var (
	configPath string
	token      string
	serverURL  string
	startTun   bool
	startHttp  bool
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
  nursor start --token your-token-here --server https://api.example.com`,
	RunE: runStart,
}

func init() {
	rootCmd.AddCommand(startCmd)

	// 配置文件路径
	startCmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to configuration file (e.g., ./config.json)")

	// Token（用于从远程获取配置）
	startCmd.Flags().StringVarP(&token, "token", "t", "", "Token for fetching configuration from remote server")

	// 远程服务器 URL（可选）
	startCmd.Flags().StringVarP(&serverURL, "server", "s", "", "Remote server URL for fetching configuration (optional)")

	// 互斥：config 和 token 不能同时使用
	startCmd.MarkFlagsMutuallyExclusive("config", "token")

	startCmd.Flags().BoolVarP(&startTun, "tun", "u", false, "Start TUN service")

	startCmd.Flags().BoolVarP(&startHttp, "http", "m", false, "Start MitmHttp service")
}

func runStart(cmd *cobra.Command, args []string) error {
	// 检查参数
	if configPath == "" && token == "" {
		return fmt.Errorf("either --config or --token must be specified")
	}

	// 方式1: 从本地文件加载配置
	if configPath != "" {
		logger.Info(fmt.Sprintf("Loading configuration from file: %s", configPath))
		if err := LoadAndApplyConfig(configPath); err != nil {
			return fmt.Errorf("failed to load config from file: %w", err)
		}
	}

	// 方式2: 从远程服务器获取配置
	if token != "" {
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
