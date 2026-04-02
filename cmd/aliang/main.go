package main

import (
	"os"

	"nursor.org/nursorgate/cmd"
)

func main() {
	handled, err := cmd.MaybeRunAsWindowsService()
	if handled {
		if err != nil {
			os.Exit(1)
		}
		return
	}

	cmd.Execute()
}
