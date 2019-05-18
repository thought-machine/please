package process

import (
	"os/exec"
	"syscall"
)

// ExecCommand executes an external command.
// We set Pdeathsig to try to make sure commands don't outlive us if we die.
func (e *Executor) ExecCommand(command string, args ...string) *exec.Cmd {
	cmd := exec.Command(command, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGHUP,
		Setpgid:   true,
	}
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.processes[cmd] = struct{}{}
	return cmd
}
