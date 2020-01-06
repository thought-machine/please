package cli

import (
	"os"
	"os/signal"
	"syscall"
)

var atexitHandlers []func()

func init() {
	go handleSignals()
}

// handleSignals waits until it receives a terminating signal from the OS, at which point it executes any
// functions previously registered with AtExit, and then exits the process.
func handleSignals() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGABRT, syscall.SIGTERM)
	sig := <-ch
	log.Info("Received signal %s", sig)
	// Allow a second signal to terminate the process regardless
	done := make(chan bool)
	go func() {
		for _, h := range atexitHandlers {
			h()
		}
		close(done)
	}()
	select {
	case <-done:
		log.Info("All exit handlers run, shutting down process")
		exit(sig)
	case sig := <-ch:
		log.Warning("Received second signal %s, aborting", sig)
		exit(sig)
	}
}

// AtExit registers a function to be run when the process is killed by a signal.
// Note that this is best-effort; we cannot guarantee that there are not other ways of exiting that
// bypass any mechanism we use here.
func AtExit(f func()) {
	atexitHandlers = append(atexitHandlers, f)
}

// exit kills the process with an exit code suitable for the given signal.
func exit(sig os.Signal) {
	if s, ok := sig.(syscall.Signal); ok {
		os.Exit(128 + int(s))
	}
	os.Exit(1)
}
