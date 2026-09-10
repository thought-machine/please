//go:build !windows
// +build !windows

package process

import "syscall"

// ExecReplace replaces the currently running process with the given command.
// It does not return unless the exec itself failed.
func ExecReplace(path string, argv, env []string) error {
	return syscall.Exec(path, argv, env)
}
