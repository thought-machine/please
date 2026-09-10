package cli

import (
	"os"
	"syscall"
)

// terminatingSignals are the signals we clean up and exit on. Windows only ever delivers
// Ctrl-C as os.Interrupt and a synthesised SIGTERM; the others are defined but never sent.
var terminatingSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}

// exitCodeForSignal returns the exit code to use when dying to a signal. The 128+signum
// convention is a shell idiom with no meaning on Windows, so just report a plain failure.
func exitCodeForSignal(sig os.Signal) int {
	return 1
}
