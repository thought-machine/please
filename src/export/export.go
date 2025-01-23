// Package export handles exporting parts of the repo to other directories.
// This is useful if, for example, one wanted to separate out part of
// their repo with all dependencies.
package export

import (
	"os"
	"path/filepath"

	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/gc"
	"github.com/thought-machine/please/src/parse"
)

var log = logging.Log

// ToDir exports a set of targets to the given directory.
// It dies on any errors.
func ToDir(state *core.BuildState, dir string, targets []core.BuildLabel) {
	done := map[*core.BuildTarget]bool{}
	for _, target := range state.Config.Parse.PreloadSubincludes {
		for _, includeLabel := range append(state.Graph.TransitiveSubincludes(target), target) {
			export(state.Graph, dir, state.Graph.TargetOrDie(includeLabel), done)
		}
	}
	for _, target := range targets {
		export(state.Graph, dir, state.Graph.TargetOrDie(target), done)
	}
	// Now write all the build files
	packages := map[*core.Package]bool{}
	for target := range done {
		packages[state.Graph.PackageOrDie(target.Label)] = true
	}
	for pkg := range packages {
		if pkg.Name == parse.InternalPackageName {
			continue // This isn't a real package to be copied
		}
		if pkg.Subrepo != nil {
			continue // Don't copy subrepo BUILD files... they don't exist in our source tree
		}
		dest := filepath.Join(dir, pkg.Filename)
		if err := fs.RecursiveCopy(pkg.Filename, dest, 0); err != nil {
			log.Fatalf("Failed to copy BUILD file %s: %s\n", pkg.Filename, err)
		}
		// Now rewrite the unused targets out of it
		victims := []string{}
		for _, target := range pkg.AllTargets() {
			if !done[target] && !target.HasParent() {
				victims = append(victims, target.Label.Name)
			}
		}
		if err := gc.RewriteFile(state, dest, victims); err != nil {
			log.Fatalf("Failed to rewrite BUILD file: %s\n", err)
		}
	}
	// Write any preloaded build defs as well; preloaded subincludes should be fine though.
	for _, preload := range state.Config.Parse.PreloadBuildDefs {
		if err := fs.RecursiveCopy(preload, filepath.Join(dir, preload), 0); err != nil {
			log.Fatalf("Failed to copy preloaded build def %s: %s", preload, err)
		}
	}

	exportPlzConf(dir)
}

func exportPlzConf(destDir string) {
	profiles, err := filepath.Glob(".plzconfig*")
	if err != nil {
		log.Fatalf("failed to glob .plzconfig files: %v", err)
	}
	for _, file := range profiles {
		path := filepath.Join(destDir, file)
		if err := os.RemoveAll(path); err != nil {
			log.Fatalf("failed to copy .plzconfig file %s: %v", file, err)
		}
		if err := fs.CopyFile(file, path, 0); err != nil {
			log.Fatalf("failed to copy .plzconfig file %s: %v", file, err)
		}
	}
}

// exportSources exports any source files (srcs and data) for the rule
func exportSources(graph *core.BuildGraph, dir string, target *core.BuildTarget) {
	for _, src := range append(target.AllSources(), target.AllData()...) {
		if _, ok := src.Label(); !ok { // We'll handle these dependencies later
			for _, p := range src.FullPaths(graph) {
				if !filepath.IsAbs(p) { // Don't copy system file deps.
					if err := fs.RecursiveCopy(p, filepath.Join(dir, p), 0); err != nil {
						log.Fatalf("Error copying file: %s\n", err)
					}
				}
			}
		}
	}
}

// export implements the logic of ToDir, but prevents repeating targets.
func export(graph *core.BuildGraph, dir string, target *core.BuildTarget, done map[*core.BuildTarget]bool) {
	if done[target] {
		return
	}
	// We want to export the package that made this subrepo available, but we still need to walk the target deps
	// as it may depend on other subrepos or first party targets
	if target.Subrepo != nil {
		export(graph, dir, target.Subrepo.Target, done)
	} else {
		exportSources(graph, dir, target)
	}

	done[target] = true
	for _, dep := range target.Dependencies() {
		export(graph, dir, dep, done)
	}
	for _, subinclude := range graph.PackageOrDie(target.Label).AllSubincludes(graph) {
		export(graph, dir, graph.TargetOrDie(subinclude), done)
	}
	if parent := target.Parent(graph); parent != nil && parent != target {
		export(graph, dir, parent, done)
	}
}

// Outputs exports the outputs of a target.
func Outputs(state *core.BuildState, dir string, targets []core.BuildLabel) {
	for _, label := range targets {
		target := state.Graph.TargetOrDie(label)
		for _, out := range target.Outputs() {
			fullPath := filepath.Join(dir, out)
			outDir := filepath.Dir(fullPath)
			if err := os.MkdirAll(outDir, core.DirPermissions); err != nil {
				log.Fatalf("Failed to create export dir %s: %s", outDir, err)
			}
			if err := fs.RecursiveCopy(filepath.Join(target.OutDir(), out), fullPath, target.OutMode()|0200); err != nil {
				log.Fatalf("Failed to copy export file: %s", err)
			}
		}
	}
}
