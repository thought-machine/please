// +build !linux

package core

import "os/exec"

// ExecCommand executes an external command.
func ExecCommand(command string, args ...string) *exec.Cmd {
	return exec.Command(command, args...)
}
