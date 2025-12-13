package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// Global flags for root command (persistent across all subcommands)
var (
	configPath string
	token      string
)

var rootCmd = &cobra.Command{
	Use:   "nonelane",
	Short: "Nonelane is a tool for managing your nonelane server",
	Long:  `Nonelane is a tool for managing your nonelane server`,
	// PersistentPreRunE: 移除，因为逻辑应该在 RunE 或子命令中处理
	// 这样可以避免在子命令执行时也执行不必要的逻辑
	RunE: func(cmd *cobra.Command, args []string) error {
		// 如果没有子命令，直接调用 runStart 的逻辑
		// 这样可以避免代码重复，统一使用 runStart 函数
		return runStart(cmd, args)
	},
}

func init() {
	// 配置文件路径
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Path to configuration file (e.g., ./config.json)")

	// Token（用于激活用户）
	rootCmd.PersistentFlags().StringVarP(&token, "token", "t", "", "Token for user activation")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
