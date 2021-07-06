package exec

import (
	"context"
	"math"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/process"
)

var log = logging.MustGetLogger("exec")

func Exec(state *core.BuildState, label core.BuildLabel, cmdArgs []string, shareNetwork bool) {
	exec(context.Background(), state, label, cmdArgs, shareNetwork)
}

func exec(ctx context.Context, state *core.BuildState, label core.BuildLabel, cmdArgs []string, shareNetwork bool) error {
	target := state.Graph.TargetOrDie(label)

	if !target.IsBinary && len(cmdArgs) == 0 {
		log.Fatalf("Target %s cannot be executed; it's not marked as binary", target.Label)
	}

	var cmd string
	if len(cmdArgs) > 0 {
		var err error
		if cmd, err = core.ReplaceSequences(state, target, strings.Join(cmdArgs, " ")); err != nil {
			return err
		}
	} else {
		outs := target.Outputs()
		if len(outs) != 1 {
			log.Fatalf("Target %s cannot be executed as it has %d outputs", target.Label, len(outs))
		}
		cmd = outs[0]
	}

	dir := filepath.Join(core.OutDir, "exec", target.Label.Subrepo, target.Label.PackageName)
	if err := PrepareRuntimeDir(state, target, dir); err != nil {
		return err
	}

	env := core.ExecEnvironment(state, target, path.Join(core.RepoRoot, dir))
	_, _, err := state.ProcessExecutor.ExecWithTimeoutShellStdStreams(target, dir, env, time.Duration(math.MaxInt64), false, process.NewSandboxConfig(!shareNetwork, true), cmd, true)

	return err
}

// TODO(tiagovtristao): We might want to find a better way of reusing this logic, since it's similarly used in a couple of places already.
func PrepareRuntimeDir(state *core.BuildState, target *core.BuildTarget, dir string) error {
	if err := fs.ForceRemove(state.ProcessExecutor, dir); err != nil {
		return err
	}

	if err := os.MkdirAll(dir, fs.DirPermissions); err != nil {
		return err
	}

	if err := state.EnsureDownloaded(target); err != nil {
		return err
	}

	for out := range core.IterRuntimeFiles(state.Graph, target, true, dir) {
		if err := core.PrepareSourcePair(out); err != nil {
			return err
		}
	}

	return nil
}
