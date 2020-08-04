package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var atexitHandlers []func()

func init() {
	go handleSignals()
	TargetLogger.run()
}

type stringable interface {
	String() string
}


type targetState struct {
	label stringable
	state string
}
type tLogger struct {
	c chan *targetState
	done chan string
	targetStatuses map[string]string
	doneTargets map[string]struct{}
}

// TODO(jpoole): delete this when done
var TargetLogger = &tLogger{
	c: make(chan *targetState, 100),
	done: make(chan string, 100),
	targetStatuses: map[string]string{},
	doneTargets: map[string]struct{}{},
}

func (t *tLogger) run() {
	go func() {
		for msg := range t.c {
			t.targetStatuses[msg.label.String()] = msg.state
		}
	}()

	go func() {
		for label := range t.done {
			t.doneTargets[label] = struct {}{}
		}
	}()
}

func (t *tLogger) Log(label stringable, msg string, args ...interface{}) {
	t.c <- &targetState{
		label: label,
		state: fmt.Sprintf(msg, args...),
	}
}

func (t *tLogger) PrintState(){
	time.Sleep(time.Second)
	log.Warningf("Done:")
	for k, _ := range t.doneTargets {
		log.Warningf(k)
	}
	log.Warningf("Pending: ")
	for k, v := range t.targetStatuses {
		if _, ok := t.doneTargets[k]; ok {
			continue
		}
		log.Warningf("%v %v", k, v)
	}
}

func(t *tLogger) Done(label stringable) {
	t.done <- label.String()
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

	TargetLogger.PrintState()

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
