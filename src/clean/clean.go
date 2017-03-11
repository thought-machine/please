// Code for cleaning Please build artifacts.

package clean

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"syscall"

	"gopkg.in/op/go-logging.v1"

	"build"
	"core"
	"test"
)

var log = logging.MustGetLogger("clean")

// Clean cleans the entire output directory and optionally the cache as well.
func Clean(config *core.Configuration, cleanCache, background bool) {
	if background {
		if err := maybeFork(core.OutDir, config.Cache.Dir, cleanCache); err != nil {
			log.Warning("Couldn't run clean in background; will do it synchronously: %s", err)
		}
	}
	if cleanCache {
		clean(config.Cache.Dir)
	}
	clean(core.OutDir)
}

// CleanTargets cleans a given set of build targets.
func CleanTargets(state *core.BuildState, labels []core.BuildLabel, cleanCache bool) {
	for _, label := range labels {
		// Clean any and all sub-targets of this target.
		// This is not super efficient; we potentially repeat this walk multiple times if
		// we have several targets to clean in a package. It's unlikely to be a big concern though
		// unless we have lots of targets to clean and their packages are very large.
		for _, target := range state.Graph.PackageOrDie(label.PackageName).AllChildren(state.Graph.TargetOrDie(label)) {
			if target.ShouldInclude(state.Include, state.Exclude) {
				cleanTarget(state, target, cleanCache)
			}
		}
	}
}

func cleanTarget(state *core.BuildState, target *core.BuildTarget, cleanCache bool) {
	if err := build.RemoveOutputs(target); err != nil {
		log.Fatalf("Failed to remove output: %s", err)
	}
	if target.IsTest {
		if err := test.RemoveCachedTestFiles(target); err != nil {
			log.Fatalf("Failed to remove file: %s", err)
		}
	}
	if cleanCache && state.Cache != nil {
		state.Cache.Clean(target)
	}
}

func clean(path string) {
	if core.PathExists(path) {
		log.Info("Cleaning path %s", path)
		if err := os.RemoveAll(path); err != nil {
			log.Fatalf("Failed to clean path %s: %s", path, err)
		}
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
	args := []string{rm, "-rf", newOutDir}
	if cleanCache {
		newCacheDir, err := moveDir(cacheDir)
		if err != nil {
			return err
		}
		args = append(args, newCacheDir)
	}
	// Note that we can't fork() directly and continue running Go code, but ForkExec() works okay.
	_, err = syscall.ForkExec(rm, args, nil)
	if err == nil {
		// Success if we get here.
		fmt.Println("Cleaning in background; you may continue to do pleasing things in this repo in the meantime.")
		os.Exit(0)
	}
	return err
}

// moveDir moves a directory to a new location and returns that new location.
func moveDir(dir string) (string, error) {
	b := make([]byte, 16)
	rand.Read(b)
	name := path.Join(path.Dir(dir), ".plz_clean_"+hex.EncodeToString(b))
	log.Notice("Moving %s to %s", dir, name)
	return name, os.Rename(dir, name)
}
