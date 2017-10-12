// Code for cleaning Please build artifacts.

package clean

import (
	"fmt"
	"os"

	"gopkg.in/op/go-logging.v1"

	"build"
	"core"
	"test"
)

var log = logging.MustGetLogger("clean")

// Clean cleans the entire output directory and optionally the cache as well.
func Clean(config *core.Configuration, cache core.Cache, background bool) {
	if cache != nil {
		cache.CleanAll()
	}
	if background {
		if err := core.AsyncDeleteDir(core.OutDir); err != nil {
			log.Warning("Couldn't run clean in background; will do it synchronously: %s", err)
		} else {
			fmt.Println("Cleaning in background; you may continue to do pleasing things in this repo in the meantime.")
			return
		}
	}
	clean(core.OutDir)
}

// Targets cleans a given set of build targets.
func Targets(state *core.BuildState, labels []core.BuildLabel, cleanCache bool) {
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
