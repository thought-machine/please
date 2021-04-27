// Package run implements the "plz run" command.
package run

import (
	"context"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/output"
	"github.com/thought-machine/please/src/process"
)

var log = logging.MustGetLogger("run")

// Run implements the running part of 'plz run'.
func Run(state *core.BuildState, label core.AnnotatedOutputLabel, args []string, remote, env, inTmp bool, dir string) {
	prepareRun(dir, inTmp)

	run(context.Background(), state, label, args, false, false, remote, env, false, inTmp, dir)
}

// Parallel runs a series of targets in parallel.
// Returns a relevant exit code (i.e. if at least one subprocess exited unsuccessfully, it will be
// that code, otherwise 0 if all were successful).
// The given context can be used to control the lifetime of the subprocesses.
func Parallel(ctx context.Context, state *core.BuildState, labels []core.AnnotatedOutputLabel, args []string, numTasks int, quiet, remote, env, detach, inTmp bool, dir string) int {
	prepareRun(dir, inTmp)

	limiter := make(chan struct{}, numTasks)
	var g errgroup.Group
	for _, label := range labels {
		label := label // capture locally
		g.Go(func() error {
			limiter <- struct{}{}
			defer func() { <-limiter }()
			return run(ctx, state, label, args, true, quiet, remote, env, detach, inTmp, dir)
		})
	}
	if err := g.Wait(); err != nil {
		if ctx.Err() != context.Canceled { // Don't error if the context killed the process.
			log.Error("Command failed: %s", err)
			return err.(*exitError).code
		}
		return 0
	}
	return 0
}

// Sequential runs a series of targets sequentially.
// Returns a relevant exit code (i.e. if at least one subprocess exited unsuccessfully, it will be
// that code, otherwise 0 if all were successful).
func Sequential(state *core.BuildState, labels []core.AnnotatedOutputLabel, args []string, quiet, remote, env, inTmp bool, dir string) int {
	prepareRun(dir, inTmp)
	for _, label := range labels {
		log.Notice("Running %s", label)
		if err := run(context.Background(), state, label, args, true, quiet, remote, env, false, inTmp, dir); err != nil {
			log.Error("%s", err)
			return err.(*exitError).code
		}
	}
	return 0
}

func prepareRun(dir string, inTmp bool) {
	if err := os.RemoveAll("plz-out/run"); err != nil && !os.IsNotExist(err) {
		log.Warningf("failed to clean up old run working directory: %v", err)
	}
}

// run implements the internal logic about running a target.
// If fork is true then we fork to run the target and return any error from the subprocesses.
// If it's false this function never returns (because we either win or die; it's like
// Game of Thrones except rather less glamorous).
func run(ctx context.Context, state *core.BuildState, label core.AnnotatedOutputLabel, args []string, fork, quiet, remote, setenv, detach, tmpDir bool, dir string) error {
	// This is a bit strange as normally if you run a binary for another platform, this will fail. In some cases
	// this can be quite useful though e.g. to compile a binary for a target arch, then run an .sh script to
	// push that to docker.
	if state.TargetArch != cli.HostArch() {
		label.Subrepo = state.TargetArch.String()
	}

	target := state.Graph.TargetOrDie(label.BuildLabel)
	if !target.IsBinary {
		log.Fatalf("Target %s cannot be run; it's not marked as binary", label)
	}
	if label.Annotation == "" && len(target.Outputs()) != 1 {
		log.Fatalf("Targets %s cannot be run as it has %d outputs.", label, len(target.Outputs()))
	}
	if remote {
		// Send this off to be done remotely.
		// This deliberately misses the out_exe bit below, but also doesn't pick up whatever's going on with java -jar;
		// however that will be obsolete post #920 anyway.
		if state.RemoteClient == nil {
			log.Fatalf("You must configure remote execution to use plz run --remote")
		}
		return state.RemoteClient.Run(target)
	}

	if tmpDir {
		var err error
		if dir, err = prepareRunDir(state, target); err != nil {
			return err
		}
	}

	// ReplaceSequences always quotes stuff in case it contains spaces or special characters,
	// that works fine if we interpret it as a shell but not to pass it as an argument here.
	command := ""
	if label.Annotation != "" {
		entryPoint, ok := target.EntryPoints[label.Annotation]
		if !ok {
			log.Fatalf("Cannot run %s as has no entry point %s", label, label.Annotation)
		}
		if tmpDir {
			command = filepath.Join(dir, entryPoint)
		} else {
			command = filepath.Join(target.OutDir(), entryPoint)
		}
	} else {
		// out_exe handles java binary stuff by invoking the .jar with java as necessary
		if tmpDir {
			command = filepath.Join(dir, target.Outputs()[0])
		} else {
			command, _ = core.ReplaceSequences(state, target, fmt.Sprintf("$(out_exe %s)", target.Label))
		}
	}
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
	} else if dir != "" { // Find an absolute path before changing directory
		abs, err := filepath.Abs(splitCmd[0])
		if err != nil {
			log.Fatalf("Couldn't calculate absolute path for %s: %s", splitCmd[0], err)
		}
		splitCmd[0] = abs
	}
	args = append(splitCmd, args...)
	log.Info("Running target %s...", strings.Join(args, " "))
	output.SetWindowTitle("plz run: " + strings.Join(args, " "))
	env := environ(state, target, setenv, tmpDir)

	if !fork {
		if dir != "" {
			err := syscall.Chdir(dir)
			if err != nil {
				log.Fatalf("Error changing directory %s: %s", dir, err)
			}
		}
		// Plain 'plz run'. One way or another we never return from the following line.
		must(syscall.Exec(splitCmd[0], args, env), args)
	} else if detach {
		// Bypass the whole process management system since we explicitly aim not to manage this subprocess.
		cmd := exec.Command(splitCmd[0], args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = dir
		return toExitError(cmd.Start(), args, nil)
	}
	// Run as a normal subcommand.
	// Note that we don't connect stdin. It doesn't make sense for multiple processes.
	// The process executor doesn't actually support not having a timeout, but the max is ~290 years so nobody
	// should know the difference.
	_, output, err := process.New("").ExecWithTimeout(ctx, nil, dir, env, time.Duration(math.MaxInt64), false, false, !quiet, args)
	return toExitError(err, args, output)
}

func prepareRunDir(state *core.BuildState, target *core.BuildTarget) (string, error) {
	path := filepath.Join("plz-out", "run", target.Label.Subrepo, target.Label.PackageName)
	if err := os.MkdirAll(path, fs.DirPermissions); err != nil && !os.IsExist(err) {
		return "", err
	}

	path, err := ioutil.TempDir(path, target.Label.Name+"_*")
	if err != nil {
		return "", err
	}

	if err := state.EnsureDownloaded(target); err != nil {
		return "", err
	}

	for out := range core.IterRuntimeFiles(state.Graph, target, true, path) {
		if err := core.PrepareSourcePair(out); err != nil {
			return "", err
		}
	}

	return path, nil
}

// environ returns an appropriate environment for a command.
func environ(state *core.BuildState, target *core.BuildTarget, setenv, tmpDir bool) []string {
	env := os.Environ()
	for _, e := range adRunEnviron {
		env = addEnv(env, e)
	}
	if setenv || tmpDir {
		for _, e := range core.RunEnvironment(state, target, tmpDir) {
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
func toExitError(err error, cmd []string, out []byte) error {
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
