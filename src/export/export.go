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
)

var log = logging.Log

// Exporter defines the interface for exporting parts of a Please repository to a new directory.
// It handles the copying of configuration files, preloaded build definitions, and selected
// targets along with their necessary source files and dependencies.
type Exporter interface {
	// ExportPlzConfig exports the repository's configuration files (e.g., .plzconfig and its
	// platform-specific variants) to the target export directory.
	ExportPlzConfig()
	// ExportPreloaded exports all globally preloaded build definitions and subincluded targets.
	// These are usually defined in the repository's configuration file.
	ExportPreloaded()
	// ExportTargets exports the set of targets identified by the given build labels.
	// Each target recursively exports all their source files and required build statements, but also
	// targets in their transitive dependencies.
	ExportTargets(core.BuildLabels)
	// ExportTarget exports an individual build target.
	// Each target recursively exports all their source files and required build statements, but also
	// targets in their transitive dependencies.
	ExportTarget(*core.BuildTarget)
	// WritePackageFiles writes the processed BUILD files for all exported targets to the
	// export directory. These BUILD files may be modified (e.g., trimmed) depending on
	// the exporter's implementation.
	WritePackageFiles()
}

// Repo export a new please repo including the targets and dependencies requested. Depending on the
// noTrim flag, the export will attempt to trim the resulting repository, exporting only the required
// targets and build statements in their packages. If noTrim is set, all targets of a package will be
// exported and not build statement trimming will be attempted, the BUILD file is copied in its entirety.
func Repo(state *core.BuildState, dir string, noTrim bool, targets []core.BuildLabel) {
	e := newExporter(state, dir, noTrim)

	// ensure output dir
	if err := os.MkdirAll(dir, fs.DirPermissions); err != nil {
		log.Fatalf("failed to create export directory %s: %v", dir, err)
	}

	e.ExportPlzConfig()
	e.ExportPreloaded()
	e.ExportTargets(targets)
	e.WritePackageFiles()
}

// Outputs exports the build artifacts (output files) produced by building the specified
// targets to the given output directory.
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

// newExporter creates a new exporter of a specific type based on the arguments.
func newExporter(state *core.BuildState, dir string, noTrim bool) Exporter {
	base := baseExporter{
		state:           state,
		targetDir:       dir,
		exportedTargets: map[core.BuildLabel]bool{},
	}

	if noTrim {
		exporter := &noTrimExporter{
			baseExporter:     base,
			exportedPackages: map[string]bool{},
		}
		exporter.impl = exporter
		return exporter
	}

	exporter := &defaultExporter{
		baseExporter:         base,
		visitedPackages:      map[core.BuildLabel]bool{},
		requiredSubincludes:  map[core.BuildLabel]core.BuildLabels{},
		preloadedSubincludes: map[core.BuildLabel]bool{},
	}
	exporter.impl = exporter
	return exporter
}

// baseExporter provides common fields and methods of other exporters.
type baseExporter struct {
	state     *core.BuildState
	targetDir string

	// exportedTargets maintains a record of the targets that have been exported so far.
	exportedTargets map[core.BuildLabel]bool
	// impl is a reference to the concrete exporter implementation. It's included for calling the
	// specific exporter implementation from the common methods.
	impl Exporter
}

func (be *baseExporter) ExportPlzConfig() {
	profiles, err := filepath.Glob(".plzconfig*")
	if err != nil {
		log.Fatalf("failed to glob .plzconfig files: %v", err)
	}
	for _, file := range profiles {
		targetPath := filepath.Join(be.targetDir, file)
		if err := os.RemoveAll(targetPath); err != nil {
			log.Fatalf("failed to remove .plzconfig file %s: %v", file, err)
		}
		if err := fs.CopyFile(file, targetPath, 0); err != nil {
			log.Fatalf("failed to copy .plzconfig file %s: %v", file, err)
		}
	}
}

func (be *baseExporter) ExportTargets(labels core.BuildLabels) {
	for _, l := range labels {
		target := be.getOrParseTarget(l)
		if target == nil {
			log.Errorf("Unable to lookup target %s", l)
			continue
		}
		be.impl.ExportTarget(target)
	}
}

func (be *baseExporter) getOrParseTarget(label core.BuildLabel) *core.BuildTarget {
	target := be.state.Graph.Target(label)
	if target == nil {
		log.Debugf("Target %v not found in graph. Attempting to parse...", label)
		be.state.WaitForBuiltTarget(label, core.OriginalTarget, core.ParseModeNormal)
		target = be.state.Graph.Target(label)
	}
	return target
}

// exportDependencies exports exportDependencies of a target.
func (be *baseExporter) exportDependencies(target *core.BuildTarget) {
	deps := target.DeclaredDependencies()
	log.Debugf("Exporting dependencies of (%v): %v", target.Label, deps)
	be.ExportTargets(deps)
}

// exportSources exports all files required by the target.
func (be *baseExporter) exportSources(target *core.BuildTarget) {
	for _, src := range append(target.AllSources(), target.AllData()...) {
		if _, ok := src.Label(); ok {
			continue // These will be handled as dependencies later
		}
		for _, p := range src.Paths(be.state.Graph) {
			if filepath.IsAbs(p) { // Don't copy system file deps.
				log.Infof("System dependency detected, skipping...: %s", p)
				continue
			}
			if err := fs.RecursiveCopy(p, filepath.Join(be.targetDir, p), 0); err != nil {
				log.Warningf("Error copying file, skipping...: %s", err)
			}
			log.Debugf("Writing exported source file: %s", p)
		}
	}
}

// checkAndSetVisited is a helper to ensure we only visit the same target once.
// It returns true if this is the first time the target is being exported.
func (be *baseExporter) checkAndSetVisited(target *core.BuildTarget) bool {
	visited := be.exportedTargets[target.Label]
	be.exportedTargets[target.Label] = true
	return !visited
}
