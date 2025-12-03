package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"nursor.org/nursorgate/common/logger"
	runnerUtils "nursor.org/nursorgate/inbound/tun/runner/utils"
	proxyConfig "nursor.org/nursorgate/processor/config"
	proxyRegistry "nursor.org/nursorgate/processor/proxy"
)

// 添加其他有用的命令

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Long:  `Print the version number of nursor`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("nursor v1.0.0")
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management commands",
	Long:  `Manage configuration: load, save, validate, etc.`,
}

var configLoadCmd = &cobra.Command{
	Use:   "load [config-file]",
	Short: "Load configuration from file",
	Long:  `Load configuration from a JSON file and apply it to the system`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := args[0]
		logger.Info(fmt.Sprintf("Loading configuration from: %s", configPath))
		return LoadAndApplyConfig(configPath)
	},
}

var configFetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch configuration from remote server",
	Long:  `Fetch configuration from remote server using token`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if token == "" {
			return fmt.Errorf("--token is required for fetching config from remote")
		}
		logger.Info("Fetching configuration from remote server...")
		return FetchAndApplyConfigFromRemote(token, serverURL)
	},
}

var configSaveCmd = &cobra.Command{
	Use:   "save [output-file]",
	Short: "Save current configuration to file",
	Long:  `Save the current configuration to a JSON file`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// 从系统获取当前配置并保存
		configs := proxyConfig.GetConfigStore().GetAll()
		registry := proxyRegistry.GetRegistry()
		defaultProxy := ""
		if p, err := registry.GetDefault(); err == nil {
			defaultProxy = p.Addr()
		}

		config := &Config{
			Engine:       nil, // TODO: 从系统获取
			CurrentProxy: defaultProxy,
			CoreServer:   runnerUtils.GetServerHost(),
			Proxies:      configs,
		}
		return SaveConfigToFile(config, args[0])
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)

	configCmd.AddCommand(configLoadCmd)
	configCmd.AddCommand(configFetchCmd)
	configCmd.AddCommand(configSaveCmd)

	// config fetch 命令需要 token
	configFetchCmd.Flags().StringVarP(&token, "token", "t", "", "Token for fetching configuration")
	configFetchCmd.Flags().StringVarP(&serverURL, "server", "s", "", "Remote server URL (optional)")
}
