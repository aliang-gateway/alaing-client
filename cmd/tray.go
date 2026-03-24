package cmd

import (
	"github.com/spf13/cobra"
	"nursor.org/nursorgate/app/tray"
	"nursor.org/nursorgate/common/logger"
)

var trayCmd = &cobra.Command{
	Use:   "tray",
	Short: "Start system tray application",
	Long: `Start the aliang system tray application.

This will create a system tray icon with a menu to control the server.
Supports Linux, macOS, and Windows.

Examples:
  # Start with system tray
  aliang tray
  
  # Start with config file and tray
  aliang tray --config ./config.json`,
	RunE: runTray,
}

func init() {
	rootCmd.AddCommand(trayCmd)
}

func runTray(cmd *cobra.Command, args []string) error {
	logger.Info("Starting Aliang in system tray mode...")

	guard, acquired, err := acquireSingleInstanceGuard()
	if err != nil {
		return err
	}
	if !acquired {
		logger.Info("Aliang is already running, opening dashboard...")
		openDashboardInBrowser()
		return nil
	}
	defer guard.Close()

	// Load startup configuration using the same precedence as start command:
	// --config > ./config.new.json > ~/.aliang/config.json > database snapshot > embedded default
	if err := ApplyStartupConfig(configPath); err != nil {
		return err
	}

	// Start the tray application
	// This will block until the user quits
	tray.Run()

	return nil
}
