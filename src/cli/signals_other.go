//go:build !windows
// +build !windows

package cli

import (
	"os"
	"syscall"
)

// terminatingSignals are the signals we clean up and exit on.
var terminatingSignals = []os.Signal{
	syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGABRT, syscall.SIGTERM,
}

// exitCodeForSignal returns the conventional shell exit code for dying to a signal.
func exitCodeForSignal(sig os.Signal) int {
	if s, ok := sig.(syscall.Signal); ok {
		return 128 + int(s)
	}
	return 1
}
