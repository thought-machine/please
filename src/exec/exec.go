package exec

import (
	"fmt"
	"math"
	osExec "os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/process"
)

var log = logging.Log

// Exec allows the execution of a target or override command in a sandboxed environment that can also be configured to have some namespaces shared.
func Exec(state *core.BuildState, label core.AnnotatedOutputLabel, dir string, env, overrideCmdArgs []string, foreground bool, sandbox process.SandboxConfig) int {
	target := state.Graph.TargetOrDie(label.BuildLabel)
	if len(overrideCmdArgs) == 0 {
		if entryPoint, ok := target.EntryPoints[label.Annotation]; ok {
			overrideCmdArgs = []string{entryPoint}
		} else if label.Annotation != "" {
			log.Fatalf("%v has no such entry point %v", label.BuildLabel, label.Annotation)
		}
	}
	if err := exec(state, target, dir, env, overrideCmdArgs, foreground, sandbox); err != nil {
		log.Error("Command execution failed: %s", err)
		if exitError, ok := err.(*osExec.ExitError); ok {
			return exitError.ExitCode()
		}
		return 1
	}
	return 0
}

// exec runs the given command in the given directory, with the given environment and arguments.
func exec(state *core.BuildState, target *core.BuildTarget, runtimeDir string, env []string, overrideCmdArgs []string, foreground bool, sandbox process.SandboxConfig) error {
	if !target.IsBinary && len(overrideCmdArgs) == 0 {
		return fmt.Errorf("Either the target needs to be a binary or an override command must be provided")
	}

	if err := core.PrepareRuntimeDir(state, target, runtimeDir); err != nil {
		return err
	}

	cmd, err := resolveCmd(state, target, overrideCmdArgs, runtimeDir, sandbox)
	if err != nil {
		return err
	}

	env = append(core.ExecEnvironment(state, target, filepath.Join(core.RepoRoot, runtimeDir)), env...)
	_, _, err = state.ProcessExecutor.ExecWithTimeoutShellStdStreams(target, runtimeDir, env, time.Duration(math.MaxInt64), false, foreground, sandbox, cmd, true)
	return err
}

// resolveCmd resolves the command to run for the given target.
func resolveCmd(state *core.BuildState, target *core.BuildTarget, overrideCmdArgs []string, runtimeDir string, sandbox process.SandboxConfig) (string, error) {
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
