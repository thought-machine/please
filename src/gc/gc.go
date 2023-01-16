// Package gc implements "garbage collection" logic for Please, which is an attempt to identify
// targets in the repo that are no longer needed.
// The definition of "needed" is a bit unclear; we define it as non-test binaries, but the
// command accepts an argument to add extra ones just in case (for example, if you have a repo which
// is primarily a library, you might have to tell it that).
package gc

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/parse/asp"
	"github.com/thought-machine/please/src/scm"
)

var log = logging.Log

type targetMap map[*core.BuildTarget]bool

// GarbageCollect initiates the garbage collection logic.
func GarbageCollect(state *core.BuildState, filter, targets, keepTargets []core.BuildLabel, keepLabels []string, conservative, targetsOnly, srcsOnly, noPrompt, dryRun, git bool) {
	if targets, srcs := targetsToRemove(state.Graph, filter, targets, keepTargets, keepLabels, conservative); len(targets) > 0 {
		if !srcsOnly {
			fmt.Fprintf(os.Stderr, "Targets to remove (total %d of %d):\n", len(targets), len(state.Graph.AllTargets()))
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
		} else if !noPrompt && !cli.PromptYN("Remove these targets / files?", false) {
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
				if err := scm.NewFallback(core.RepoRoot).Remove(srcs); err != nil {
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
func targetsToRemove(graph *core.BuildGraph, filter, targets, targetsToKeep []core.BuildLabel, keepLabels []string, includeTests bool) (core.BuildLabels, []string) {
	keepTargets := targetMap{}
	for _, target := range graph.AllTargets() {
		if (target.IsBinary && (!target.IsTest() || includeTests)) || target.HasAnyLabel(keepLabels) || anyInclude(targetsToKeep, target.Label) || target.Label.Subrepo != "" {
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
					for _, target := range pkg.AllTargets() {
						log.Debug("GC root: %s", target.Label)
						addTarget(graph, keepTargets, target)
					}
				}
			}
		} else {
			addTarget(graph, keepTargets, graph.Target(target))
		}
	}
	log.Notice("%d targets to keep after configured GC roots", len(keepTargets))
	if !includeTests {
		// This is a bit complex - need to identify any tests that are tests "on" the set of things
		// we've already decided to keep.
		for _, target := range graph.AllTargets() {
			if target.IsTest() {
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
		for _, src := range target.AllLocalSourcePaths() {
			keepSrcs[src] = true
		}
	}
	ret := make(core.BuildLabels, 0, len(keepTargets))
	retSrcs := []string{}
	for _, target := range graph.AllTargets() {
		if sibling := gcSibling(graph, target); !sibling.HasParent() && !keepTargets[sibling] && isIncluded(sibling, filter) {
			ret = append(ret, target.Label)
			for _, src := range target.AllLocalSourcePaths() {
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
	if m[target] || target == nil {
		return
	}
	log.Debug("  %s", target.Label)
	m[target] = true
	for _, dep := range target.DeclaredDependencies() {
		addTarget(graph, m, graph.Target(dep))
	}
	for _, dep := range target.Dependencies() {
		addTarget(graph, m, dep)
	}
}

// anyInclude returns true if any of the given labels include this one.
func anyInclude(labels []core.BuildLabel, label core.BuildLabel) bool {
	for _, l := range labels {
		if l.Includes(label) {
			return true
		}
	}
	return false
}

// publicDependencies returns the public dependencies of a target, considering any
// private targets it might have declared.
// For example, if we have dependencies as follows:
//
//	//src/test:container_test
//	//src/test:_container_test#lib
//	//src/test:test
//
// it will return //src/test:test for //src/test:container_test.
func publicDependencies(graph *core.BuildGraph, target *core.BuildTarget) []*core.BuildTarget {
	ret := []*core.BuildTarget{}
	for _, dep := range target.DeclaredDependencies() {
		if depTarget := graph.Target(dep); depTarget != nil {
			if depTarget.Label.Parent() == target.Label.Parent() {
				ret = append(ret, publicDependencies(graph, depTarget)...)
			} else {
				ret = append(ret, depTarget)
			}
		}
	}
	return ret
}

// RewriteFile rewrites a BUILD file to exclude a set of targets.
func RewriteFile(state *core.BuildState, filename string, targets []string) error {
	p := asp.NewParser(state)
	stmts, err := p.ParseFileOnly(filename)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(filename)
	if err != nil {
		return err // This is very unlikely since we already read it once above, but y'know...
	}
	lines := bytes.Split(b, []byte{'\n'})
	linesToDelete := map[int]bool{}
	f := asp.NewFile(filename)
	for _, target := range targets {
		stmt := asp.FindTarget(stmts, target)
		if stmt == nil {
			log.Warning("Can't find target %s in %s", target, filename)
			continue
		}
		start, end := asp.GetExtents(f, stmts, stmt, len(lines))
		for i := start; i <= end; i++ {
			linesToDelete[i-1] = true // -1 because the extents are 1-indexed
		}
	}
	// Now rewrite the actual file
	lines2 := make([][]byte, 0, len(lines))
	for i, line := range lines {
		if !linesToDelete[i] {
			lines2 = append(lines2, line)
		}
	}
	return os.WriteFile(filename, bytes.Join(lines2, []byte{'\n'}), 0664)
}

// removeTargets rewrites the given set of targets out of their BUILD files.
func removeTargets(state *core.BuildState, labels core.BuildLabels) error {
	byPackage := map[*core.Package][]string{}
	for _, l := range labels {
		pkg := state.Graph.PackageOrDie(l)
		byPackage[pkg] = append(byPackage[pkg], l.Name)
	}
	for pkg, victims := range byPackage {
		log.Notice("Rewriting %s to remove %s...\n", pkg.Filename, strings.Join(victims, ", "))
		if err := RewriteFile(state, pkg.Filename, victims); err != nil {
			return err
		}
	}
	return nil
}

// gcSibling finds any labelled sibling of this target, i.e. if it says gc_sibling:target1
// then it returns target1 in the same package.
// This is for cases where multiple targets are generated by the same rule and should
// therefore share the same GC fate.
func gcSibling(graph *core.BuildGraph, t *core.BuildTarget) *core.BuildTarget {
	for _, l := range t.PrefixedLabels("gc_sibling:") {
		if t2 := graph.Target(core.NewBuildLabel(t.Label.PackageName, l)); t2 != nil {
			return t2
		}
		log.Warning("Target %s declared a gc_sibling of %s, but %s doesn't exist", t.Label, l, l)
	}
	return t
}
