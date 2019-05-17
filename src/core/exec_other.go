// +build !linux

package core

import (
	"os/exec"
	"syscall"
)

// ExecCommand executes an external command.
func ExecCommand(command string, args ...string) *exec.Cmd {
	cmd := exec.Command(command, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	return cmd
}
