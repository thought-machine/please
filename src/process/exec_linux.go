package process

import (
	"os/exec"
	"syscall"
)

// ExecCommand executes an external command.
// We set Pdeathsig to try to make sure commands don't outlive us if we die.
// N.B. This does not start the command - the caller must handle that (or use one
//      of the other functions which are higher-level interfaces).
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

// MustSandboxCommand modifies the given command to run in the sandbox.
func (e *Executor) MustSandboxCommand(cmd []string) []string {
	if e.sandboxCommand == "" {
		log.Fatalf("Sandbox tool not found on PATH")
	}
	return append([]string{e.sandboxCommand}, cmd...)
}
