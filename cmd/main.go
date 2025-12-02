package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "nursor",
	Short: "Nursor is a tool for managing your nursor server",
	Long:  `Nursor is a tool for managing your nursor server`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
