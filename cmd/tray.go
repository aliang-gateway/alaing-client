package cmd

import (
	"github.com/spf13/cobra"
	"nursor.org/nursorgate/app/tray"
	"nursor.org/nursorgate/common/logger"
)

var trayCmd = &cobra.Command{
	Use:   "tray",
	Short: "Start system tray application",
	Long: `Start the nonelane system tray application.

This will create a system tray icon with a menu to control the server.
Supports Linux, macOS, and Windows.

Examples:
  # Start with system tray
  nonelane tray
  
  # Start with config file and tray
  nonelane tray --config ./config.json`,
	RunE: runTray,
}

func init() {
	rootCmd.AddCommand(trayCmd)
}

func runTray(cmd *cobra.Command, args []string) error {
	logger.Info("Starting Nonelane in system tray mode...")
	
	// Load configuration (same as start command)
	if configPath != "" {
		logger.Info("Loading configuration from file: " + configPath)
		if err := LoadAndApplyConfig(configPath); err != nil {
			return err
		}
	}
	
	// Start the tray application
	// This will block until the user quits
	tray.Run()
	
	return nil
}