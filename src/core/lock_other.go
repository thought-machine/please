//go:build !windows
// +build !windows

package core

import (
	"os"
	"syscall"
)

const (
	lockShared      = syscall.LOCK_SH
	lockExclusive   = syscall.LOCK_EX
	lockUnlock      = syscall.LOCK_UN
	lockNonBlocking = syscall.LOCK_NB
)

// flock applies or releases an advisory lock on an open file.
func flock(file *os.File, how int) error {
	return syscall.Flock(int(file.Fd()), how)
}
