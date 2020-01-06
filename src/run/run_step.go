// Package run implements the "plz run" command.
package run

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/output"
	"github.com/thought-machine/please/src/process"
)

var log = logging.MustGetLogger("run")

// Run implements the running part of 'plz run'.
func Run(state *core.BuildState, label core.BuildLabel, args []string, env bool) {
	run(context.Background(), state, label, args, false, false, env)
}

// Parallel runs a series of targets in parallel.
// Returns a relevant exit code (i.e. if at least one subprocess exited unsuccessfully, it will be
// that code, otherwise 0 if all were successful).
// The given context can be used to control the lifetime of the subprocesses.
func Parallel(ctx context.Context, state *core.BuildState, labels []core.BuildLabel, args []string, numTasks int, quiet, env bool) int {
	pool := NewGoroutinePool(numTasks)
	var g errgroup.Group
	for _, label := range labels {
		label := label // capture locally
		g.Go(func() (err error) {
			var wg sync.WaitGroup
			wg.Add(1)
			pool.Submit(func() {
				if e := run(ctx, state, label, args, true, quiet, env); e != nil {
					err = e
				}
				wg.Done()
			})
			wg.Wait()
			return
		})
	}
	if err := g.Wait(); err != nil {
		if ctx.Err() != context.Canceled { // Don't error if the context killed the process.
			log.Error("Command failed: %s", err)
		}
		return err.(*exitError).code
	}
	return 0
}

// Sequential runs a series of targets sequentially.
// Returns a relevant exit code (i.e. if at least one subprocess exited unsuccessfully, it will be
// that code, otherwise 0 if all were successful).
func Sequential(state *core.BuildState, labels []core.BuildLabel, args []string, quiet, env bool) int {
	for _, label := range labels {
		log.Notice("Running %s", label)
		if err := run(context.Background(), state, label, args, true, quiet, env); err != nil {
			log.Error("%s", err)
			return err.code
		}
	}
	return 0
}

// run implements the internal logic about running a target.
// If fork is true then we fork to run the target and return any error from the subprocesses.
// If it's false this function never returns (because we either win or die; it's like
// Game of Thrones except rather less glamorous).
func run(ctx context.Context, state *core.BuildState, label core.BuildLabel, args []string, fork, quiet, setenv bool) *exitError {
	target := state.Graph.TargetOrDie(label)
	if !target.IsBinary {
		log.Fatalf("Target %s cannot be run; it's not marked as binary", label)
	}
	if len(target.Outputs()) != 1 && target.IsTest {
		log.Fatalf("Targets %s cannot be run as it has %d outputs; Only tests with 1 output can be run.", label, len(target.Outputs()))
	}
	// ReplaceSequences always quotes stuff in case it contains spaces or special characters,
	// that works fine if we interpret it as a shell but not to pass it as an argument here.
	command, _ := core.ReplaceSequences(state, target, fmt.Sprintf("$(out_exe %s)", target.Label))
	arg0 := strings.Trim(command, "\"")
	// Handle targets where $(exe ...) returns something nontrivial
	splitCmd := strings.Split(arg0, " ")
	if !strings.Contains(splitCmd[0], "/") {
		// Probably it's a java -jar, we need an absolute path to it.
		cmd, err := exec.LookPath(splitCmd[0])
		if err != nil {
			log.Fatalf("Can't find binary %s", splitCmd[0])
		}
		splitCmd[0] = cmd
	}
	args = append(splitCmd, args...)
	log.Info("Running target %s...", strings.Join(args, " "))
	output.SetWindowTitle("plz run: " + strings.Join(args, " "))
	env := environ(state.Config, setenv)
	if !fork {
		// Plain 'plz run'. One way or another we never return from the following line.
		must(syscall.Exec(splitCmd[0], args, env), args)
	}
	// Run as a normal subcommand.
	// Note that we don't connect stdin. It doesn't make sense for multiple processes.
	// The process executor doesn't actually support not having a timeout, but the max is ~290 years so nobody
	// should know the difference.
	_, output, err := process.New("").ExecWithTimeout(target, "", env, time.Duration(math.MaxInt64), false, false, !quiet, args)
	return toExitError(err, args, output)
}

// environ returns an appropriate environment for a command.
func environ(config *core.Configuration, setenv bool) []string {
	env := os.Environ()
	for _, e := range adRunEnviron {
		env = addEnv(env, e)
	}
	if setenv {
		for _, e := range core.GeneralBuildEnvironment(config) {
			env = addEnv(env, e)
		}
	}
	return env
}

// adRunEnviron returns values that are appended to the environment for a command.
var adRunEnviron = []string{
	"PEX_NOCACHE=true",
}

// addEnv adds an env var to an existing set, with replacement.
func addEnv(env []string, e string) []string {
	name := e[:strings.IndexRune(e, '=')+1]
	for i, existing := range env {
		if strings.HasPrefix(existing, name) {
			env[i] = e
			return env
		}
	}
	return append(env, e)
}

// must dies if the given error is non-nil.
func must(err error, cmd []string) {
	if err != nil {
		log.Fatalf("Error running command %s: %s", strings.Join(cmd, " "), err)
	}
}

// toExitError attempts to extract the exit code from an error.
func toExitError(err error, cmd []string, out []byte) *exitError {
	exitCode := 1
	if err == nil {
		return nil
	} else if exitError, ok := err.(*exec.ExitError); ok {
		// This is a little hairy; there isn't a good way of getting the exit code,
		// but this should be reasonably portable (at least to the platforms we care about).
		if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
			exitCode = status.ExitStatus()
		}
	}
	return &exitError{
		msg:  fmt.Sprintf("Error running command %s: %s\n%s", strings.Join(cmd, " "), err, string(out)),
		code: exitCode,
	}
}

type exitError struct {
	msg  string
	code int
}

func (e *exitError) Error() string {
	return e.msg
}
