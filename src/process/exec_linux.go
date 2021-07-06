package process

import (
	"os"
	"os/exec"
	"syscall"
)

type SandboxConfig struct {
	Network, Mount bool
}

var NoSandbox = SandboxConfig{}

func NewSandboxConfig(network, mount bool) SandboxConfig {
	return SandboxConfig{Network: network, Mount: mount}
}

// ExecCommand executes an external command.
// We set Pdeathsig to try to make sure commands don't outlive us if we die.
// N.B. This does not start the command - the caller must handle that (or use one
//      of the other functions which are higher-level interfaces).
func (e *Executor) ExecCommand(sandbox SandboxConfig, command string, args ...string) *exec.Cmd {
	shouldNamespace := e.namespace == NamespaceAlways || (e.namespace == NamespaceSandbox && sandbox != NoSandbox)

	// If we're sandboxing, run the sandbox tool to set up the network, mount, etc.
	var env []string
	if sandbox != NoSandbox {
		// re-exec into `plz sandbox` if we're using the built in sandboxing
		if e.usePleaseSandbox {
			if !shouldNamespace {
				log.Fatalf("can't use please sandbox and not namespace")
			}
			args = append([]string{"sandbox", command}, args...)
			plz, err := os.Executable()
			if err != nil {
				panic(err)
			}
			command = plz
		} else {
			// Otherwise exec the sandbox tool
			args = append([]string{command}, args...)
			command = e.sandboxTool
			env = []string{"SANDBOX_NETWORK=" + boolToString(sandbox.Network), "SANDBOX_MOUNT=" + boolToString(sandbox.Mount)}
		}
	}
	cmd := exec.Command(command, args...)
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGHUP,
		Setpgid:   true,
	}

	// If we have any sort of sandboxing set up, we should always namespace, however we only namespace mount and net if
	// we're sandboxing this rule.
	if shouldNamespace {
		cmd.SysProcAttr.Cloneflags = syscall.CLONE_NEWUSER | syscall.CLONE_NEWUTS | syscall.CLONE_NEWIPC | syscall.CLONE_NEWPID
		if sandbox != NoSandbox {
			if sandbox.Network {
				// Namespace network
				cmd.SysProcAttr.Cloneflags |= syscall.CLONE_NEWNET
			}
			if sandbox.Mount {
				// Namespace mount
				cmd.SysProcAttr.Cloneflags |= syscall.CLONE_NEWNS
			}
		}

		cmd.SysProcAttr.UidMappings = []syscall.SysProcIDMap{
			// Map the host user ID to 0 which is a privileged user as is treated specially by the kernel.
			// This essentially gives us fakeroot(1) style elevation for build rules
			{HostID: os.Getuid(), Size: 1, ContainerID: 0},
		}
		cmd.SysProcAttr.GidMappings = []syscall.SysProcIDMap{
			// Map the host group ID to 0. This is usually wheel or root though it doesn't have any special meaning
			// to the kernel unlike user 0
			{HostID: os.Getgid(), Size: 1, ContainerID: 0},
		}
	}
	return cmd
}

// Say nothing...
func boolToString(value bool) string {
	if value {
		return "1"
	}
	return "0"
}
