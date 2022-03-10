//go:build !windows
// +build !windows

package cli

import (
	"os"
	"os/signal"
	"syscall"
)

// notifyOnWindowResize calls the given function whenever the size of the terminal window changes
// (when receiving a SIGWINCH on most platforms).
func notifyOnWindowResize(f func()) {
	sig := make(chan os.Signal, 10)
	signal.Notify(sig, syscall.SIGWINCH)
	for range sig {
		f()
	}
}
