// +build ignore

// Package gc implements "garbage collection" logic for Please, which is an attempt to identify
// targets in the repo that are no longer needed.
// The definition of "needed" is a bit unclear; we define it as non-test binaries, but the
// command accepts an argument to add extra ones just in case (for example, if you have a repo which
// is primarily a library, you might have to tell it that).
package gc

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/Songmu/prompter"
	"gopkg.in/op/go-logging.v1"

	"core"
	"parse"
)

var log = logging.MustGetLogger("gc")

type targetMap map[*core.BuildTarget]bool

// GarbageCollect initiates the garbage collection logic.
func GarbageCollect(state *core.BuildState, filter, targets []core.BuildLabel, keepLabels []string, conservative, targetsOnly, srcsOnly, noPrompt, dryRun, git bool) {
	if targets, srcs := targetsToRemove(state.Graph, filter, targets, keepLabels, conservative); len(targets) > 0 {
		if !srcsOnly {
			fmt.Fprintf(os.Stderr, "Targets to remove (total %d of %d):\n", len(targets), state.Graph.Len())
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
		if dryRun {
			return
		} else if !noPrompt && !prompter.YN("Remove these targets / files?", false) {
			os.Exit(1)
		}
		if !srcsOnly {
			if err := removeTargets(state, targets); err != nil {
				log.Fatalf("%s\n", err)
			}
		}
		if !targetsOnly {
			if git {
				log.Notice("Running git rm %s\n", strings.Join(srcs, " "))
				srcs = append([]string{"rm", "-q"}, srcs...)
				cmd := exec.Command("git", srcs...)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					log.Fatalf("git rm failed: %s\n", err)
				}
			} else {
				for _, src := range srcs {
					log.Notice("Deleting %s...\n", src)
					if err := os.Remove(src); err != nil {
						log.Fatalf("Failed to remove %s: %s\n", src, err)
					}
				}
			}
		}
		fmt.Fprintf(os.Stderr, "Garbage collected!\n")
	} else {
		fmt.Fprintf(os.Stderr, "Nothing to remove\n")
	}
}

// targetsToRemove finds the set of targets that are no longer needed and any extraneous sources.
func targetsToRemove(graph *core.BuildGraph, filter, targets []core.BuildLabel, keepLabels []string, includeTests bool) (core.BuildLabels, []string) {
	keepTargets := targetMap{}
	for _, target := range graph.AllTargets() {
		if (target.IsBinary && (!target.IsTest || includeTests)) || target.HasAnyLabel(keepLabels) {
			log.Debug("GC root: %s", target.Label)
			addTarget(graph, keepTargets, target)
		}
	}
	// Any registered subincludes also count.
	for _, pkg := range graph.PackageMap() {
		for _, subinclude := range pkg.Subincludes {
			log.Debug("GC root: %s", subinclude)
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
						log.Debug("GC root: %s", target.Label)
						addTarget(graph, keepTargets, target)
					}
				}
			}
		} else {
			addTarget(graph, keepTargets, graph.TargetOrDie(target))
		}
	}
	log.Notice("%d targets to keep after configured GC roots", len(keepTargets))
	if !includeTests {
		// This is a bit complex - need to identify any tests that are tests "on" the set of things
		// we've already decided to keep.
		for _, target := range graph.AllTargets() {
			if target.IsTest {
				for _, dep := range publicDependencies(graph, target) {
					if keepTargets[dep] && !dep.TestOnly {
						log.Debug("Keeping test %s on %s", target.Label, dep.Label)
						addTarget(graph, keepTargets, target)
					} else if dep.TestOnly {
						log.Debug("Keeping test-only target %s", dep.Label)
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
		if !target.HasParent() && !keepTargets[target] && isIncluded(target, filter) {
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

// isIncluded returns true if the given target is included in a set of filtering labels.
func isIncluded(target *core.BuildTarget, filter []core.BuildLabel) bool {
	if len(filter) == 0 {
		return true // if you don't specify anything, the filter has no effect.
	}
	for _, f := range filter {
		if f.Includes(target.Label) {
			return true
		}
	}
	return false
}

// addTarget adds a target and all its transitive dependencies to the given map.
func addTarget(graph *core.BuildGraph, m targetMap, target *core.BuildTarget) {
	if m[target] {
		return
	}
	log.Debug("  %s", target.Label)
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

// RewriteFile rewrites a BUILD file to exclude a set of targets.
func RewriteFile(state *core.BuildState, filename string, targets []string) error {
	for i, t := range targets {
		targets[i] = fmt.Sprintf(`"%s"`, t)
	}
	data := string(MustAsset("rewrite.py"))
	// Template in the variables we want.
	data = strings.Replace(data, "__FILENAME__", filename, 1)
	data = strings.Replace(data, "__TARGETS__", strings.Replace(fmt.Sprintf("%s", targets), " ", ", ", -1), 1)
	return parse.RunCode(state, data)
}

// removeTargets rewrites the given set of targets out of their BUILD files.
func removeTargets(state *core.BuildState, labels core.BuildLabels) error {
	byPackage := map[string][]string{}
	for _, l := range labels {
		byPackage[l.PackageName] = append(byPackage[l.PackageName], l.Name)
	}
	for pkgName, victims := range byPackage {
		filename := state.Graph.PackageOrDie(pkgName).Filename
		log.Notice("Rewriting %s to remove %s...\n", filename, strings.Join(victims, ", "))
		if err := RewriteFile(state, filename, victims); err != nil {
			return err
		}
	}
	return nil
}
