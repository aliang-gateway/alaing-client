//go:build windows

package cmd

import (
	"aliang.one/nursorgate/app/tray"
	"github.com/spf13/cobra"
)

var companionCmd = &cobra.Command{
	Use:    "companion",
	Short:  "Start Windows desktop companion",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		tray.RunCompanion()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(companionCmd)
}
