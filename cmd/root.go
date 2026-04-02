package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"aliang.one/nursorgate/app/tray"
	"aliang.one/nursorgate/common/logger"
	"github.com/spf13/cobra"
)

// Global flags for root command (persistent across all subcommands)
var (
	configPath string
	token      string
)

var rootCmd = &cobra.Command{
	Use:   "aliang",
	Short: "Aliang is a tool for managing your aliang server",
	Long:  `Aliang is a tool for managing your aliang server`,
	// PersistentPreRunE: 移除，因为逻辑应该在 RunE 或子命令中处理
	// 这样可以避免在子命令执行时也执行不必要的逻辑
	RunE: func(cmd *cobra.Command, args []string) error {
		// Detect if running from .app bundle on macOS
		execPath, _ := filepath.EvalSymlinks(os.Args[0])
		if execPath != "" && strings.Contains(execPath, ".app/Contents/MacOS/") {
			// .app bundle launch → enter smart tray mode (companion)
			logger.Info("Detected .app bundle launch, starting tray mode...")
			tray.RunCompanion()
			return nil
		}
		return runDefaultRoot(cmd, args)
	},
}

func init() {
	// 配置文件路径
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Path to configuration file (e.g., ./config.json)")

	// Token（用于激活用户）
	rootCmd.PersistentFlags().StringVarP(&token, "token", "t", "", "Token for user activation")
}

func Execute() {
	defer func() {
		if recovered := recover(); recovered != nil {
			err := fmt.Errorf("panic: %v", recovered)
			logger.Error(fmt.Sprintf("Aliang panicked: %v\n%s", recovered, debug.Stack()))
			notifyExecuteError(err)
			os.Exit(1)
		}
	}()

	if err := rootCmd.Execute(); err != nil {
		logger.Error(fmt.Sprintf("Aliang command failed: %v", err))
		notifyExecuteError(err)
		os.Exit(1)
	}
}
