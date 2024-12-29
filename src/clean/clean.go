// Code for cleaning Please build artifacts.

package clean

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/thought-machine/please/src/build"
	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/test"
)

var log = logging.Log

// Clean cleans the entire output directory and optionally the cache as well.
func Clean(config *core.Configuration, cache core.Cache, background bool) {
	if cache != nil {
		cache.CleanAll()
	}
	if background {
		if err := AsyncDeleteDir(core.OutDir); err != nil {
			log.Warning("Couldn't run clean in background; will do it synchronously: %s", err)
		} else {
			fmt.Println("Cleaning in background; you may continue to do pleasing things in this repo in the meantime.")
			return
		}
	}
	clean(core.OutDir)
}

// Targets cleans a given set of build targets.
func Targets(state *core.BuildState, labels []core.BuildLabel) {
	for _, label := range labels {
		// Clean any and all sub-targets of this target.
		// This is not super efficient; we potentially repeat this walk multiple times if
		// we have several targets to clean in a package. It's unlikely to be a big concern though
		// unless we have lots of targets to clean and their packages are very large.
		for _, target := range state.Graph.PackageOrDie(label).AllChildren(state.Graph.TargetOrDie(label)) {
			if state.ShouldInclude(target) {
				cleanTarget(state, target)
			}
		}
	}
}

func cleanTarget(state *core.BuildState, target *core.BuildTarget) {
	if err := build.RemoveOutputs(target); err != nil {
		log.Fatalf("Failed to remove output: %s", err)
	}
	if target.IsTest() {
		if err := test.RemoveTestOutputs(target); err != nil {
			log.Fatalf("Failed to remove file: %s", err)
		}
	}
	if state.Cache != nil {
		state.Cache.Clean(target)
	}
}

func clean(path string) {
	if core.PathExists(path) {
		log.Info("Cleaning path %s", path)
		if err := fs.RemoveAll(path); err != nil {
			log.Fatalf("Failed to clean path %s: %s", path, err)
		}
	}
}

// AsyncDeleteDir deletes a directory asynchronously.
// First it renames the directory to something temporary and then forks to delete it.
// The rename is done synchronously but the actual deletion is async (after fork) so
// you don't have to wait for large directories to be removed.
// Conversely there is obviously no guarantee about at what point it will actually cease to
// be on disk any more.
func AsyncDeleteDir(dir string) error {
	if !fs.PathExists(dir) {
		return nil // not an error, just don't need to do anything.
	}
	exec, err := os.Executable()
	if err != nil {
		return fmt.Errorf("Failed to determine executable path: %w", err)
	}
	newDir, err := moveDir(dir)
	if err != nil {
		return err
	}
	// Note that we can't fork() directly and continue running Go code, but ForkExec() works okay,
	// so we re-execute ourselves with a specific command that will remove this.
	_, err = syscall.ForkExec(exec, []string{exec, "clean", "--rm", newDir}, nil)
	return err
}

// moveDir moves a directory to a new location and returns that new location.
func moveDir(dir string) (string, error) {
	b := make([]byte, 16)
	rand.Read(b)
	name := filepath.Join(filepath.Dir(dir), ".plz_clean_"+hex.EncodeToString(b))
	log.Notice("Moving %s to %s", dir, name)
	return name, os.Rename(dir, name)
}
