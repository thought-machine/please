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
func GarbageCollect(graph *core.BuildGraph, targets []core.BuildLabel, keepLabels []string, conservative, targetsOnly, srcsOnly bool) {
	if targets, srcs := targetsToRemove(graph, targets, keepLabels, conservative); len(targets) > 0 {
		if !srcsOnly {
			fmt.Fprintf(os.Stderr, "Targets to remove (total %d of %d):\n", len(targets), graph.Len())
			for _, target := range targets {
				fmt.Printf("  %s\n", target)
			}
		}
		if !targetsOnly && len(srcs) > 0 {
			fmt.Fprintf(os.Stderr, "Corresponding source files to remove:\n")
			for _, src := range srcs {
				fmt.Printf("  %s\n", src)
			}
		}
	} else {
		fmt.Fprintf(os.Stderr, "Nothing to remove\n")
	}
}

// targetsToRemove finds the set of targets that are no longer needed and any extraneous sources.
func targetsToRemove(graph *core.BuildGraph, targets []core.BuildLabel, keepLabels []string, includeTests bool) (core.BuildLabels, []string) {
	keepTargets := targetMap{}
	for _, target := range graph.AllTargets() {
		if (target.IsBinary && (!target.IsTest || includeTests)) || target.HasAnyLabel(keepLabels) {
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
				for _, dep := range publicDependencies(graph, target) {
					if keepTargets[dep] {
						addTarget(graph, keepTargets, target)
					} else if dep.TestOnly {
						addTarget(graph, keepTargets, dep)
					}
				}
			}
		}
		log.Notice("%d targets to keep after exploring tests", len(keepTargets))
	}
	// Now build the set of sources that we'll keep. This is important because other targets that
	// we're not deleting could still use the sources of the targets that we are.
	keepSrcs := map[string]bool{}
	for target := range keepTargets {
		for _, src := range target.AllLocalSources() {
			keepSrcs[src] = true
		}
	}
	ret := make(core.BuildLabels, 0, len(keepTargets))
	retSrcs := []string{}
	for _, target := range graph.AllTargets() {
		if !target.HasParent() && !keepTargets[target] {
			ret = append(ret, target.Label)
			for _, src := range target.AllLocalSources() {
				if !keepSrcs[src] {
					retSrcs = append(retSrcs, src)
				}
			}
		}
	}
	sort.Sort(ret)
	sort.Strings(retSrcs)
	log.Notice("%d targets to remove", len(ret))
	log.Notice("%d sources to remove", len(retSrcs))
	return ret, retSrcs
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
	for _, dep := range target.Dependencies() {
		addTarget(graph, m, dep)
	}
}

// publicDependencies returns the public dependencies of a target, considering any
// private targets it might have declared.
// For example, if we have dependencies as follows:
//   //src/test:container_test
//   //src/test:_container_test#lib
//   //src/test:test
// it will return //src/test:test for //src/test:container_test.
func publicDependencies(graph *core.BuildGraph, target *core.BuildTarget) []*core.BuildTarget {
	ret := []*core.BuildTarget{}
	for _, dep := range target.DeclaredDependencies() {
		depTarget := graph.TargetOrDie(dep)
		if depTarget.Label.Parent() == target.Label.Parent() {
			ret = append(ret, publicDependencies(graph, depTarget)...)
		} else {
			ret = append(ret, depTarget)
		}
	}
	return ret
}
