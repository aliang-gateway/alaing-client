package cmd

import (
	"github.com/spf13/cobra"
	"nursor.org/nursorgate/app/tray"
	"nursor.org/nursorgate/common/logger"
)

var trayAgentCmd = &cobra.Command{
	Use:    "tray-agent",
	Short:  "Start the menu bar companion for a background service",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger.Info("Starting Aliang menu bar companion...")
		tray.RunCompanion()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(trayAgentCmd)
}
