package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"nursor.org/nursorgate/common/logger"
)

// 添加其他有用的命令

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Long:  `Print the version number of nonelane`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("nonelane v1.0.0")
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

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)

	configCmd.AddCommand(configLoadCmd)
}
