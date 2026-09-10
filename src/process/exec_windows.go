package process

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// ExecCommand executes an external command.
// Windows has no process groups in the POSIX sense; the closest equivalent is a console
// process group, which is what CREATE_NEW_PROCESS_GROUP sets up. That gives us somewhere to
// send Ctrl-Break, which is the nearest thing to SIGTERM. Killing the whole tree is handled
// separately by a job object - see kill_windows.go.
//
// N.B. This does not start the command - the caller must handle that (or use one
// of the other functions which are higher-level interfaces).
func (e *Executor) ExecCommand(sandbox SandboxConfig, foreground bool, command string, args ...string) *exec.Cmd {
	// There is no sandboxing on Windows yet; sandbox and foreground are both ignored.
	cmd := exec.Command(command, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP,
	}
	return cmd
}
