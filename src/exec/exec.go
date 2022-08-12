package exec

import (
	"fmt"
	"math"
	osExec "os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/process"
)

var log = logging.Log

// Exec allows the execution of a target or override command in a sandboxed environment that can also be configured to have some namespaces shared.
func Exec(state *core.BuildState, label core.AnnotatedOutputLabel, dir string, env, overrideCmdArgs []string, foreground bool, sandbox process.SandboxConfig) int {
	target := state.Graph.TargetOrDie(label.BuildLabel)
	return exitCode(exec(state, process.Default, target, dir, env, overrideCmdArgs, nil, label.Annotation, foreground, sandbox))
}

// Sequential executes a series of targets in sequence, stopping when one fails.
// It returns the exit code from the last executed target; if that's zero then they were all successful.
func Sequential(state *core.BuildState, outputMode process.OutputMode, labels []core.AnnotatedOutputLabel, args []string, shareNetwork, shareMount bool) int {
	for _, label := range labels {
		target := state.Graph.TargetOrDie(label.BuildLabel)
		sandbox := process.NewSandboxConfig(target.Sandbox && !shareNetwork, target.Sandbox && !shareMount)
		if err := exec(state, outputMode, target, target.ExecDir(), nil, nil, args, label.Annotation, false, sandbox); err != nil {
			return exitCode(err)
		}
	}
	return 0
}

// Parallel executes a series of targets in parallel (to a limit of simultaneous processes).
// Returns a relevant exit code (i.e. if at least one subprocess exited unsuccessfully, it will be
// that code, otherwise 0 if all were successful).
func Parallel(state *core.BuildState, outputMode process.OutputMode, labels []core.AnnotatedOutputLabel, args []string, numTasks int, shareNetwork, shareMount bool) int {
	limiter := make(chan struct{}, numTasks)
	var g errgroup.Group

	for _, label := range labels {
		target := state.Graph.TargetOrDie(label.BuildLabel)
		annotation := label.Annotation
		g.Go(func() error {
			limiter <- struct{}{}
			defer func() { <-limiter }()

			log.Notice("Executing %s...", target)
			sandbox := process.NewSandboxConfig(target.Sandbox && !shareNetwork, target.Sandbox && !shareMount)
			return exec(state, outputMode, target, target.ExecDir(), nil, nil, args, annotation, false, sandbox)
		})
	}
	return exitCode(g.Wait())
}

// exitCode extracts an exit code from an error, if possible.
func exitCode(err error) int {
	if err != nil {
		if ee, ok := err.(*osExec.ExitError); ok {
			return ee.ExitCode()
		}
		return 1
	}
	return 0
}

// exec runs the given command in the given directory, with the given environment and arguments.
func exec(state *core.BuildState, outputMode process.OutputMode, target *core.BuildTarget, runtimeDir string, env []string, overrideCmdArgs, additionalArgs []string, entrypoint string, foreground bool, sandbox process.SandboxConfig) error {
	if err := process.RunWithOutput(outputMode, target.Label.String(), func() ([]byte, error) {
		if !target.IsBinary && len(overrideCmdArgs) == 0 {
			return nil, fmt.Errorf("Either the target needs to be a binary or an override command must be provided")
		}

		if err := core.PrepareRuntimeDir(state, target, runtimeDir); err != nil {
			return nil, err
		}

		cmd, err := resolveCmd(state, target, overrideCmdArgs, entrypoint, runtimeDir, sandbox)
		if err != nil {
			return nil, err
		}
		if len(additionalArgs) != 0 {
			cmd += " " + strings.Join(additionalArgs, " ")
		}

		env = append(core.ExecEnvironment(state, target, filepath.Join(core.RepoRoot, runtimeDir)), env...)
		out, _, err := state.ProcessExecutor.ExecWithTimeoutShellStdStreams(target, runtimeDir, env, time.Duration(math.MaxInt64), false, foreground, sandbox, cmd, outputMode == process.Default)
		return out, err
	}); err != nil {
		log.Error("Failed to execute %s: %s", target, err)
		return err
	}
	return nil
}

// resolveCmd resolves the command to run for the given target.
func resolveCmd(state *core.BuildState, target *core.BuildTarget, overrideCmdArgs []string, entrypoint string, runtimeDir string, sandbox process.SandboxConfig) (string, error) {
	if entrypoint != "" {
		if ep, ok := target.EntryPoints[entrypoint]; ok {
			overrideCmdArgs = []string{ep}
		} else {
			log.Fatalf("%v has no such entry point %v", target, entrypoint)
		}
	}
	// The override command takes precedence if provided
	if len(overrideCmdArgs) > 0 {
		return core.ReplaceSequences(state, target, strings.Join(overrideCmdArgs, " "))
	}

	outs := target.Outputs()
	if len(outs) != 1 {
		return "", fmt.Errorf("Target %s cannot be executed as it has %d outputs", target.Label, len(outs))
	}

	if !sandbox.Mount {
		return filepath.Join(core.RepoRoot, runtimeDir, outs[0]), nil
	}
	return filepath.Join(core.SandboxDir, outs[0]), nil
}
