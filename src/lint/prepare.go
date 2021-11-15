package lint

import (
	"os"

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

// Symlinks the source files of this rule into its temp directory.
func prepareSources(state *core.BuildState, graph *core.BuildGraph, target *core.BuildTarget, tmpDir string) error {
	for source := range core.IterSources2(state, graph, target, false, tmpDir) {
		if err := core.PrepareSourcePair(source); err != nil {
			return err
		}
	}
	return nil
}
