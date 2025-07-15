//go:build windows

package utils

import (
	"os/exec"
	"syscall"
)

func RunCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
	return cmd.Run()
}

func GetRunCommand(name string, args ...string) *exec.Cmd {
	if len(args) > 0 && name == "powershell" && args[0] == "-Command" {
		args[1] = "[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new();" + args[1]
	}
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true, // 隐藏命令行窗口
	}
	return cmd
}
