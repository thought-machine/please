//go:build !windows
// +build !windows

package process

import (
	"os/exec"
	"syscall"
)

// trackProcessTree records a started process so that its descendants can be killed later.
// On Unix the process group set up in ExecCommand is sufficient, so this is a no-op.
func trackProcessTree(cmd *exec.Cmd) {}

// untrackProcessTree releases any resources held by trackProcessTree.
func untrackProcessTree(cmd *exec.Cmd) {}

// killProcessTree signals a process and all of its descendants.
func killProcessTree(cmd *exec.Cmd, sig syscall.Signal) error {
	// Kill the group - we always set one in ExecCommand.
	return syscall.Kill(-cmd.Process.Pid, sig)
}
