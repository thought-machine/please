//go:build !windows
// +build !windows

package clean

import "os/exec"

// startDetached starts a process that will outlive us, and does not wait for it.
func startDetached(bin string, args []string) error {
	return exec.Command(bin, args...).Start()
}
