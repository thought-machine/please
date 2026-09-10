package clean

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// startDetached starts a process that will outlive us, and does not wait for it.
// DETACHED_PROCESS keeps it off our console, so it isn't killed when we exit or when the
// user closes the window.
func startDetached(bin string, args []string) error {
	cmd := exec.Command(bin, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.DETACHED_PROCESS}
	return cmd.Start()
}
