// Package export handles exporting parts of the repo to other directories.
// This is useful if, for example, one wanted to separate out part of
// their repo with all dependencies.
package export

import (
	"path"
	"strings"

	"gopkg.in/op/go-logging.v1"

	"core"
	"gc"
)

var log = logging.MustGetLogger("export")

// ToDir exports a set of targets to the given directory.
// It dies on any errors.
func ToDir(state *core.BuildState, dir string, targets []core.BuildLabel) {
	done := map[*core.BuildTarget]bool{}
	for _, target := range targets {
		export(state.Graph, dir, state.Graph.TargetOrDie(target), done)
	}
	// Now write all the build files
	packages := map[*core.Package]bool{}
	for target := range done {
		packages[state.Graph.PackageOrDie(target.Label.PackageName)] = true
	}
	for pkg := range packages {
		dest := path.Join(dir, pkg.Filename)
		if err := core.RecursiveCopyFile(pkg.Filename, dest, 0, false, false); err != nil {
			log.Fatalf("Failed to copy BUILD file: %s\n", pkg.Filename)
		}
		// Now rewrite the unused targets out of it
		victims := []string{}
		for name, target := range pkg.Targets {
			if !done[target] {
				victims = append(victims, name)
			}
		}
		if err := gc.RewriteFile(state, dest, victims); err != nil {
			log.Fatalf("Failed to rewrite BUILD file: %s\n", err)
		}
	}
}

// export implements the logic of ToDir, but prevents repeating targets.
func export(graph *core.BuildGraph, dir string, target *core.BuildTarget, done map[*core.BuildTarget]bool) {
	if done[target] {
		return
	}
	for _, src := range target.AllSources() {
		if src.Label() == nil { // We'll handle these dependencies later
			for _, p := range src.FullPaths(graph) {
				if !strings.HasPrefix(p, "/") { // Don't copy system file deps.
					if err := core.RecursiveCopyFile(p, path.Join(dir, p), 0, false, false); err != nil {
						log.Fatalf("Error copying file: %s\n", err)
					}
				}
			}
		}
	}
	done[target] = true
	for _, dep := range target.Dependencies() {
		if parent := dep.Parent(graph); parent != nil && parent != target.Parent(graph) && parent != target {
			export(graph, dir, parent, done)
		} else {
			export(graph, dir, dep, done)
		}
	}
	for _, subinclude := range graph.PackageOrDie(target.Label.PackageName).Subincludes {
		export(graph, dir, graph.TargetOrDie(subinclude), done)
	}
}
