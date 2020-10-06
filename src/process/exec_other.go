// +build !linux
// +build !windows

package process

import (
	"os/exec"
	"syscall"
)

// ExecCommand executes an external command.
// N.B. This does not start the command - the caller must handle that (or use one
//      of the other functions which are higher-level interfaces).
func (e *Executor) ExecCommand(command string, args ...string) *exec.Cmd {
	cmd := exec.Command(command, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.processes[cmd] = struct{}{}
	return cmd
}

// MustSandboxCommand modifies the given command to run in the sandbox.
// On non-Linux platforms this is a no-op since namespaces aren't available.
func (e *Executor) MustSandboxCommand(cmd []string) []string {
	return cmd
}

// Kill will kill a process with the given signal
func Kill(pid int, sig syscall.Signal) error {
	return syscall.Kill(pid, sig)
}

// ForkExec will run the process asyc
func ForkExec(cmd string, args []string ) error {
	_, err := syscall.ForkExec(cmd, args, nil)
	return err
}
