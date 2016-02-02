// Code for cleaning Please build artifacts.

package clean

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"syscall"

	"github.com/op/go-logging"

	"core"
	"test"
)

var log = logging.MustGetLogger("clean")

// Clean cleans the output directory and optionally the cache as well.
// If labels is non-empty then only those specific targets will be wiped, otherwise
// everything will be removed.
func Clean(state *core.BuildState, labels []core.BuildLabel, cleanCache, background bool) {
	if len(labels) == 0 {
		if background {
			if err := maybeFork(core.OutDir, state.Config.Cache.Dir, cleanCache); err != nil {
				log.Warning("Couldn't run clean in background; will do it synchronously: %s", err)
			}
		}
		if cleanCache {
			clean(state.Config.Cache.Dir)
		}
		clean(core.OutDir)
	} else {
		for _, label := range labels {
			cleanTarget(state, state.Graph.TargetOrDie(label), cleanCache)
		}
	}
}

func cleanTarget(state *core.BuildState, target *core.BuildTarget, cleanCache bool) {
	for _, out := range target.Outputs() {
		clean(path.Join(target.OutDir(), out))
	}
	if target.IsTest {
		if err := test.RemoveCachedTestFiles(target); err != nil {
			log.Fatalf("Failed to remove file: %s", err)
		}
	}
	if cleanCache && state.Cache != nil {
		(*state.Cache).Clean(target)
	}
}

func clean(what string) {
	dir := path.Join(core.RepoRoot, what)
	if !core.PathExists(dir) {
		// Out path doesn't exist. Nothing to do.
		log.Info("Path %s not found, nothing to do.", dir)
		return
	}
	log.Info("Cleaning path %s", dir)
	if err := os.RemoveAll(dir); err != nil {
		log.Fatalf("Failed to clean path %s: %s", dir, err)
	}
}

// maybeFork will fork & detach if background is true. First it will rename the out and
// cache dirs so it's safe to run another plz in this repo, then fork & detach child
// processes to do the actual cleaning.
// The parent will then die quietly and the children will continue to actually remove the
// directories.
func maybeFork(outDir, cacheDir string, cleanCache bool) error {
	rm, err := exec.LookPath("rm")
	if err != nil {
		return err
	}
	if !core.PathExists(outDir) || !core.PathExists(cacheDir) {
		return nil
	}
	newOutDir, err := moveDir(outDir)
	if err != nil {
		return err
	}
	newCacheDir, err := moveDir(cacheDir)
	if err != nil {
		return err
	}
	// Note that we can't fork() directly and continue running Go code, but ForkExec() works okay.
	_, err = syscall.ForkExec(rm, []string{rm, "-rf", newOutDir, newCacheDir}, nil)
	if err == nil {
		// Success if we get here.
		fmt.Println("Cleaning in background; you may continue to do pleasing things in this repo in the meantime.")
		os.Exit(0)
	}
	return err
}

// moveDir moves a directory to a new location and returns that new location.
func moveDir(dir string) (string, error) {
	name, err := ioutil.TempDir(filepath.Dir(dir), ".plz_clean_")
	if err != nil {
		return "", err
	}
	log.Notice("Moving %s to %s", dir, name)
	return name, os.Rename(dir, name)
}
	
