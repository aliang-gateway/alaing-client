//go:build !windows

package cmd

import "github.com/spf13/cobra"

func runDefaultRoot(cmd *cobra.Command, args []string) error {
	return runCommandLineDefaultRoot(cmd, args)
}
