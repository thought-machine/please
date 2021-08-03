package exec

import (
	"fmt"
	"math"
	"os"
	osExec "os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/process"
)

var log = logging.MustGetLogger("exec")

// Exec allows the execution of a target or override command in a sandboxed environment that can also be configured to have some namespaces shared.
func Exec(state *core.BuildState, label core.BuildLabel, overrideCmdArgs []string, sandbox process.SandboxConfig) int {
	target := state.Graph.TargetOrDie(label)
	if err := exec(state, target, overrideCmdArgs, sandbox); err != nil {
		log.Error("Command execution failed: %s", err)
		if exitError, ok := err.(*osExec.ExitError); ok {
			return exitError.ExitCode()
		}
		return 1
	}
	return 0
}

func exec(state *core.BuildState, target *core.BuildTarget, overrideCmdArgs []string, sandbox process.SandboxConfig) error {
	if !target.IsBinary && len(overrideCmdArgs) == 0 {
		return fmt.Errorf("Either the target needs to be a binary or an override command must be provided")
	}

	runtimeDir, err := prepareRuntimeDir(state, target)
	if err != nil {
		return err
	}

	cmd, err := resolveCmd(state, target, overrideCmdArgs, runtimeDir, sandbox)
	if err != nil {
		return err
	}

	env := core.ExecEnvironment(state, target, runtimeDir)
	_, _, err = state.ProcessExecutor.ExecWithTimeoutShellStdStreams(target, runtimeDir, env, time.Duration(math.MaxInt64), false, sandbox, cmd, true)
	return err
}

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

// TODO(tiagovtristao): We might want to find a better way of reusing this logic, since it's similarly used in a couple of places already.
func prepareRuntimeDir(state *core.BuildState, target *core.BuildTarget) (string, error) {
	dir := filepath.Join(core.OutDir, "exec", target.Label.Subrepo, target.Label.PackageName)

	if err := fs.ForceRemove(state.ProcessExecutor, dir); err != nil {
		return dir, err
	}

	if err := os.MkdirAll(dir, fs.DirPermissions); err != nil {
		return dir, err
	}

	if err := state.EnsureDownloaded(target); err != nil {
		return dir, err
	}

	for out := range core.IterRuntimeFiles(state.Graph, target, true, dir) {
		if err := core.PrepareSourcePair(out); err != nil {
			return dir, err
		}
	}

	return dir, nil
}
