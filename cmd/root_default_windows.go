//go:build windows

package cmd

import (
	"aliang.one/nursorgate/app/tray"
	"github.com/spf13/cobra"
)

func runDefaultRoot(cmd *cobra.Command, args []string) error {
	tray.RunCompanion()
	return nil
}
