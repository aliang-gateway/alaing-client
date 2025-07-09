//go:build unix

package utils

import (
	"os/exec"
)

func RunCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)

	return cmd.Run()
}

func GetRunCommand(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)

	return cmd
}
