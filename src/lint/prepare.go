package lint

import (
	"fmt"
	"os"
	"path"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

// TODO(peterebden): Somehow unify a bunch of this stuff between build/test/lint

func prepareDirectory(state *core.BuildState, directory string) error {
	if core.PathExists(directory) {
		if err := fs.ForceRemove(state.ProcessExecutor, directory); err != nil {
			return err
		}
	}
	return os.MkdirAll(directory, core.DirPermissions)
}

// Links the source files of this rule into its temp directory.
func prepareSources(state *core.BuildState, graph *core.BuildGraph, target *core.BuildTarget, tmpDir string) error {
	for source := range core.IterSources2(state, graph, target, false, tmpDir) {
		if err := core.PrepareSourcePair(source); err != nil {
			return err
		}
	}
	return nil
}

// Links the output files of this rule into a temp directory.
func prepareOutputs(state *core.BuildState, target *core.BuildTarget, tmpDir string) error {
	outDir := target.OutDir()
	for _, out := range target.Outputs() {
		if err := core.PrepareSourcePair(core.SourcePair{Src: path.Join(outDir, out), Tmp: path.Join(tmpDir, out)}); err != nil {
			return err
		}
	}
	return nil
}

// command returns the command we'd run for a linter.
func command(graph *core.BuildGraph, linter *core.Linter) (string, error) {
	cmd := linter.Cmd + " $SRCS"
	if linter.Target.IsEmpty() {
		return cmd, nil
	}
	outs := graph.TargetOrDie(linter.Target).Outputs()
	if len(outs) == 0 {
		return "", fmt.Errorf("Target %s cannot be used as a linter, it has no outputs", linter.Target)
	}
	return outs[0] + " " + cmd, nil
}
