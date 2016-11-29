// Package gc implements "garbage collection" logic for Please, which is an attempt to identify
// targets in the repo that are no longer needed.
// The definition of "needed" is a bit unclear; we define it as non-test binaries, but the
// command accepts an argument to add extra ones just in case (for example, if you have a repo which
// is primarily a library, you might have to tell it that).
// Note that right now it doesn't do anything to actually "collect" the garbage, i.e. it tells
// you what to do but doesn't rewrite BUILD files itself.
package gc

import (
	"fmt"
	"os"
	"sort"

	"gopkg.in/op/go-logging.v1"

	"core"
)

var log = logging.MustGetLogger("gc")

type targetMap map[*core.BuildTarget]bool

// GarbageCollect initiates the garbage collection logic.
func GarbageCollect(graph *core.BuildGraph, targets []core.BuildLabel, conservative bool) {
	if targets := targetsToRemove(graph, targets, conservative); len(targets) > 0 {
		fmt.Fprintf(os.Stderr, "Targets to remove (total %d of %d):\n", len(targets), graph.Len())
		for _, target := range targets {
			fmt.Printf("  %s\n", target)
		}
		fmt.Fprintf(os.Stderr, "Corresponding source files to remove:\n")
		for _, target := range targets {
			for _, src := range graph.TargetOrDie(target).AllSources() {
				// Make sure we only check local file labels (not system files or anything)
				if file, ok := src.(core.FileLabel); ok {
					fmt.Printf("  %s\n", file.Paths(graph)[0])
				}
			}
		}
	} else {
		fmt.Fprintf(os.Stderr, "Nothing to remove\n")
	}
}

// targetsToRemove finds the set of targets that are no longer needed.
func targetsToRemove(graph *core.BuildGraph, targets []core.BuildLabel, includeTests bool) core.BuildLabels {
	keepTargets := targetMap{}
	for _, target := range graph.AllTargets() {
		if target.IsBinary && (!target.IsTest || includeTests) {
			addTarget(graph, keepTargets, target)
		}
	}
	// Any registered subincludes also count.
	for _, pkg := range graph.PackageMap() {
		for _, subinclude := range pkg.Subincludes {
			addTarget(graph, keepTargets, graph.TargetOrDie(subinclude))
		}
	}
	log.Notice("%d targets to keep from initial scan", len(keepTargets))
	for _, target := range targets {
		if target.IsAllSubpackages() {
			// For slightly awkward reasons these can't be handled outside :(
			for _, pkg := range graph.PackageMap() {
				if pkg.IsIncludedIn(target) {
					for _, target := range pkg.Targets {
						addTarget(graph, keepTargets, target)
					}
				}
			}
		} else {
			addTarget(graph, keepTargets, graph.TargetOrDie(target))
		}
	}
	log.Notice("%d targets to keep after command-line arguments considered", len(keepTargets))
	if !includeTests {
		// This is a bit complex - need to identify any tests that are tests "on" the set of things
		// we've already decided to keep.
		for _, target := range graph.AllTargets() {
			if target.IsTest {
				for _, dep := range target.DeclaredDependencies() {
					depTarget := graph.TargetOrDie(dep)
					if keepTargets[depTarget] {
						addTarget(graph, keepTargets, target)
					} else if depTarget.TestOnly {
						addTarget(graph, keepTargets, depTarget)
					}
				}
			}
		}
		log.Notice("%d targets to keep after exploring tests", len(keepTargets))
	}
	ret := make(core.BuildLabels, 0, len(keepTargets))
	for _, target := range graph.AllTargets() {
		if !target.HasParent() && !keepTargets[target] {
			ret = append(ret, target.Label)
		}
	}
	sort.Sort(ret)
	log.Notice("%d targets to remove", len(ret))
	return ret
}

// addTarget adds a target and all its transitive dependencies to the given map.
func addTarget(graph *core.BuildGraph, m targetMap, target *core.BuildTarget) {
	if m[target] {
		return
	}
	m[target] = true
	for _, dep := range target.DeclaredDependencies() {
		addTarget(graph, m, graph.TargetOrDie(dep))
	}
}
