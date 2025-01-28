// Package export handles exporting parts of the repo to other directories.
// This is useful if, for example, one wanted to separate out part of
// their repo with all dependencies.
package export

import (
	iofs "io/fs"
	"os"
	"path/filepath"

	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/gc"
	"github.com/thought-machine/please/src/parse"
)

var log = logging.Log

type export struct {
	state     *core.BuildState
	targetDir string
	noTrim    bool

	exportedTargets  map[core.BuildLabel]bool
	exportedPackages map[string]bool
}

// ToDir exports a set of targets to the given directory.
// It dies on any errors.
func ToDir(state *core.BuildState, dir string, noTrim bool, targets []core.BuildLabel) {
	e := &export{
		state:            state,
		noTrim:           noTrim,
		targetDir:        dir,
		exportedPackages: map[string]bool{},
		exportedTargets:  map[core.BuildLabel]bool{},
	}

	e.exportPlzConf()
	for _, target := range state.Config.Parse.PreloadSubincludes {
		for _, includeLabel := range append(state.Graph.TransitiveSubincludes(target), target) {
			e.export(state.Graph.TargetOrDie(includeLabel))
		}
	}
	for _, target := range targets {
		e.export(state.Graph.TargetOrDie(target))
	}
	// Now write all the build files
	packages := map[*core.Package]bool{}
	for target := range e.exportedTargets {
		packages[state.Graph.PackageOrDie(target)] = true
	}

	// Write any preloaded build defs as well; preloaded subincludes should be fine though.
	for _, preload := range state.Config.Parse.PreloadBuildDefs {
		if err := fs.RecursiveCopy(preload, filepath.Join(dir, preload), 0); err != nil {
			log.Fatalf("Failed to copy preloaded build def %s: %s", preload, err)
		}
	}

	if noTrim {
		return // We have already exported the whole directory
	}

	for pkg := range packages {
		if pkg.Name == parse.InternalPackageName {
			continue // This isn't a real package to be copied
		}
		if pkg.Subrepo != nil {
			continue // Don't copy subrepo BUILD files... they don't exist in our source tree
		}
		dest := filepath.Join(dir, pkg.Filename)
		if err := fs.CopyFile(pkg.Filename, dest, 0); err != nil {
			log.Fatalf("Failed to copy BUILD file %s: %s\n", pkg.Filename, err)
		}
		// Now rewrite the unused targets out of it
		var victims []string
		for _, target := range pkg.AllTargets() {
			if !e.exportedTargets[target.Label] && !target.HasParent() {
				victims = append(victims, target.Label.Name)
			}
		}
		if err := gc.RewriteFile(state, dest, victims); err != nil {
			log.Fatalf("Failed to rewrite BUILD file: %s\n", err)
		}
	}
}

func (e *export) exportPlzConf() {
	profiles, err := filepath.Glob(".plzconfig*")
	if err != nil {
		log.Fatalf("failed to glob .plzconfig files: %v", err)
	}
	for _, file := range profiles {
		path := filepath.Join(e.targetDir, file)
		if err := os.RemoveAll(path); err != nil {
			log.Fatalf("failed to copy .plzconfig file %s: %v", file, err)
		}
		if err := fs.CopyFile(file, path, 0); err != nil {
			log.Fatalf("failed to copy .plzconfig file %s: %v", file, err)
		}
	}
}

// exportSources exports any source files (srcs and data) for the rule
func (e *export) exportSources(target *core.BuildTarget) {
	for _, src := range append(target.AllSources(), target.AllData()...) {
		if _, ok := src.Label(); !ok { // We'll handle these dependencies later
			for _, p := range src.FullPaths(e.state.Graph) {
				if !filepath.IsAbs(p) { // Don't copy system file deps.
					if err := fs.RecursiveCopy(p, filepath.Join(e.targetDir, p), 0); err != nil {
						log.Fatalf("Error copying file: %s\n", err)
					}
				}
			}
		}
	}
}

// exportPackage exports the package BUILD file and all sources
func (e *export) exportPackage(pkgName string) {
	if pkgName == parse.InternalPackageName {
		return
	}
	if e.exportedPackages[pkgName] {
		return
	}
	e.exportedPackages[pkgName] = true

	err := filepath.WalkDir(pkgName, func(path string, d iofs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != pkgName && fs.IsPackage(e.state.Config.Parse.BuildFileName, path) {
				return filepath.SkipDir // We want to stop when we find another package in our dir tree
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil // Ignore symlinks, which are almost certainly generated sources.
		}
		dest := filepath.Join(e.targetDir, path)
		if err := fs.EnsureDir(dest); err != nil {
			return err
		}
		return fs.CopyFile(path, dest, 0)
	})
	if err != nil {
		log.Fatalf("failed to export package: %v", err)
	}
}

// export implements the logic of ToDir, but prevents repeating targets.
func (e *export) export(target *core.BuildTarget) {
	if e.exportedTargets[target.Label] {
		return
	}
	// We want to export the package that made this subrepo available, but we still need to walk the target deps
	// as it may depend on other subrepos or first party targets
	if target.Subrepo != nil {
		e.export(target.Subrepo.Target)
	} else if e.noTrim {
		// Export the whole package, rather than trying to trim the package down to only the targets we need
		e.exportPackage(target.Label.PackageName)
	} else {
		e.exportSources(target)
	}

	e.exportedTargets[target.Label] = true
	for _, dep := range target.Dependencies() {
		e.export(dep)
	}
	for _, subinclude := range e.state.Graph.PackageOrDie(target.Label).AllSubincludes(e.state.Graph) {
		e.export(e.state.Graph.TargetOrDie(subinclude))
	}
	if parent := target.Parent(e.state.Graph); parent != nil && parent != target {
		e.export(parent)
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
