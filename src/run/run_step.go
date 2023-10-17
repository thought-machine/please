// Package run implements the "plz run" command.
package run

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/output"
	"github.com/thought-machine/please/src/process"
)

var log = logging.Log

// Run implements the running part of 'plz run'.
func Run(state *core.BuildState, label core.AnnotatedOutputLabel, args []string, remote, env, inTmp bool, dir, overrideCmd string) {
	prepareRun()

	run(context.Background(), state, label, args, false, false, remote, env, false, inTmp, dir, overrideCmd)
}

// Parallel runs a series of targets in parallel.
// Returns a relevant exit code (i.e. if at least one subprocess exited unsuccessfully, it will be
// that code, otherwise 0 if all were successful).
// The given context can be used to control the lifetime of the subprocesses.
func Parallel(ctx context.Context, state *core.BuildState, labels []core.AnnotatedOutputLabel, args []string, numTasks int, outputMode process.OutputMode, remote, env, detach, inTmp bool, dir string) int {
	prepareRun()

	var g errgroup.Group
	g.SetLimit(numTasks)
	for _, label := range labels {
		label := label // capture locally
		g.Go(func() error {
			err := runWithOutput(ctx, state, label, args, outputMode, remote, env, detach, inTmp, dir)
			if err != nil && ctx.Err() == nil {
				log.Error("Command failed: %s", err)
			}
			return err
		})
	}
	if err := g.Wait(); err != nil {
		if ctx.Err() != nil { // Don't error if the context killed the process.
			return 0
		}
		return err.(*exitError).code
	}
	return 0
}

// runWithOutput runs a subprocess with the given output mechanism.
func runWithOutput(ctx context.Context, state *core.BuildState, label core.AnnotatedOutputLabel, args []string, outputMode process.OutputMode, remote, env, detach, inTmp bool, dir string) error {
	return process.RunWithOutput(outputMode, label.String(), func() ([]byte, error) {
		out, _, err := run(ctx, state, label, args, true, outputMode != process.Default, remote, env, detach, inTmp, dir, "")
		return out, err
	})
}

// Sequential runs a series of targets sequentially.
// Returns a relevant exit code (i.e. if at least one subprocess exited unsuccessfully, it will be
// that code, otherwise 0 if all were successful).
func Sequential(state *core.BuildState, labels []core.AnnotatedOutputLabel, args []string, outputMode process.OutputMode, remote, env, inTmp bool, dir string) int {
	prepareRun()
	for _, label := range labels {
		log.Notice("Running %s", label)
		if err := runWithOutput(context.Background(), state, label, args, outputMode, remote, env, false, inTmp, dir); err != nil {
			log.Error("%s", err)
			return err.(*exitError).code
		}
	}
	return 0
}

func prepareRun() {
	if err := os.RemoveAll("plz-out/run"); err != nil && !os.IsNotExist(err) {
		log.Warningf("failed to clean up old run working directory: %v", err)
	}
}

// run implements the internal logic about running a target.
// If fork is true then we fork to run the target and return any error from the subprocesses.
// If it's false this function never returns (because we either win or die; it's like
// Game of Thrones except rather less glamorous).
func run(ctx context.Context, state *core.BuildState, label core.AnnotatedOutputLabel, args []string, fork, quiet, remote, setenv, detach, tmpDir bool, dir, overrideCmd string) ([]byte, []byte, error) {
	// This is a bit strange as normally if you run a binary for another platform, this will fail. In some cases
	// this can be quite useful though e.g. to compile a binary for a target arch, then run an .sh script to
	// push that to docker.
	if state.TargetArch != cli.HostArch() {
		label.Subrepo = state.TargetArch.String()
	}

	target := state.Graph.TargetOrDie(label.BuildLabel)
	// Non binary targets can be run if an override command is passed in
	if !target.IsBinary && overrideCmd == "" {
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
		return nil, nil, state.RemoteClient.Run(target)
	}

	if tmpDir {
		var err error
		if dir, err = prepareRunDir(state, target); err != nil {
			return nil, nil, err
		}
	}

	// ReplaceSequences always quotes stuff in case it contains spaces or special characters,
	// that works fine if we interpret it as a shell but not to pass it as an argument here.
	switch {
	case overrideCmd != "":
		command, _ := core.ReplaceSequences(state, target, overrideCmd)
		// We don't care about passed in args when an override command is provided
		args = process.BashCommand("bash", strings.Trim(command, "\""), true)
	case label.Annotation != "":
		entryPoint, ok := target.EntryPoints[label.Annotation]
		if !ok {
			log.Fatalf("Cannot run %s as has no entry point %s", label, label.Annotation)
		}
		var command string
		if tmpDir {
			command = filepath.Join(dir, entryPoint)
		} else {
			command = filepath.Join(target.OutDir(), entryPoint)
		}
		args = append(strings.Split(command, " "), args...)
	default:
		// out_exe handles java binary stuff by invoking the .jar with java as necessary
		var command string
		if tmpDir {
			command = filepath.Join(dir, target.Outputs()[0])
		} else {
			command, _ = core.ReplaceSequences(state, target, fmt.Sprintf("$(out_exe %s)", target.Label))
			command = strings.Trim(command, "\"")
		}
		args = append(strings.Split(command, " "), args...)
	}

	// Handle targets where $(exe ...) returns something nontrivial
	if !strings.Contains(args[0], "/") {
		// Probably it's a java -jar, we need an absolute path to it.
		cmd, err := exec.LookPath(args[0])
		if err != nil {
			log.Fatalf("Can't find binary %s", args[0])
		}
		args[0] = cmd
	} else if dir != "" { // Find an absolute path before changing directory
		abs, err := filepath.Abs(args[0])
		if err != nil {
			log.Fatalf("Couldn't calculate absolute path for %s: %s", args[0], err)
		}
		args[0] = abs
	}

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
		must(syscall.Exec(args[0], args, env), args)
	} else if detach {
		// Bypass the whole process management system since we explicitly aim not to manage this subprocess.
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = dir
		return nil, nil, toExitError(cmd.Start(), args, nil)
	}
	// Run as a normal subcommand.
	// Note that we don't connect stdin. It doesn't make sense for multiple processes.
	// The process executor doesn't actually support not having a timeout, but the max is ~290 years so nobody
	// should know the difference.
	out, combined, err := process.New().ExecWithTimeout(ctx, nil, dir, env, time.Duration(math.MaxInt64), false, false, !quiet, false, process.NoSandbox, args)
	return out, combined, toExitError(err, args, combined)
}

func prepareRunDir(state *core.BuildState, target *core.BuildTarget) (string, error) {
	path := filepath.Join("plz-out", "run", target.Label.Subrepo, target.Label.PackageName)
	if err := os.MkdirAll(path, fs.DirPermissions); err != nil && !os.IsExist(err) {
		return "", err
	}

	path, err := os.MkdirTemp(path, target.Label.Name+"_*")
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
