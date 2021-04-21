package process

import (
	"os"
	"os/exec"
	"syscall"
)

// ExecCommand executes an external command.
// We set Pdeathsig to try to make sure commands don't outlive us if we die.
// N.B. This does not start the command - the caller must handle that (or use one
//      of the other functions which are higher-level interfaces).
func (e *Executor) ExecCommand(namespaceNet, namespaceMount bool, command string, args ...string) *exec.Cmd {
	cmd := exec.Command(command, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGHUP,
		Setpgid:   true,
	}

	if e.shouldNamespace {
		cmd.SysProcAttr.Cloneflags = syscall.CLONE_NEWUSER|syscall.CLONE_NEWUTS|syscall.CLONE_NEWIPC|syscall.CLONE_NEWPID
		if namespaceNet {
			cmd.SysProcAttr.Cloneflags = cmd.SysProcAttr.Cloneflags|syscall.CLONE_NEWNET
		}
		if namespaceMount {
			cmd.SysProcAttr.Cloneflags = cmd.SysProcAttr.Cloneflags|syscall.CLONE_NEWNS
		}

		cmd.SysProcAttr.UidMappings = []syscall.SysProcIDMap{
			// Map the host user ID to 0 which is a privileged user as is treated specially by the kernel.
			// This essentially gives us fakeroot(1) style elevation for build rules
			{0, os.Getuid(), 1},
		}
		cmd.SysProcAttr.GidMappings = []syscall.SysProcIDMap{
			// Map the host group ID to 0. This is usually wheel or root though it doesn't have any special meaning
			// to the kernel unlike user 0
			{0, os.Getgid(), 1},
		}
	}

	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.processes[cmd] = struct{}{}
	return cmd
}

