//go:build windows

package cmd

import (
	"aliang.one/nursorgate/app/tray"
	"github.com/spf13/cobra"
)

func runDefaultRoot(cmd *cobra.Command, args []string) error {
	writeStartupTrace("runDefaultRoot entering Windows companion by default")
	tray.RunWindowsCompanion()
	writeStartupTrace("runDefaultRoot returned from Windows companion")
	return nil
}
