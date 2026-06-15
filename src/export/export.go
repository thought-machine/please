// Package export handles exporting parts of the repo to other directories.
// This is useful if, for example, one wanted to separate out part of
// their repo with all dependencies.
package export

import (
	"os"
	"path/filepath"
	"time"

	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/parse"
)

var log = logging.Log

// Repo export a new please repo including the targets and dependencies requested. Depending on the
// noTrim flag, the export will attempt to trim the resulting repository, exporting only the required
// targets and build statements in their packages. If noTrim is set, all targets of a package will be
// exported and no build statement trimming will be attempted, the BUILD file is copied in its entirety.
func Repo(state *core.BuildState, dir string, noTrim bool, targets []core.BuildLabel) {
	e := newExporter(state, dir, noTrim)

	// ensure output dir
	if err := os.MkdirAll(dir, fs.DirPermissions); err != nil {
		log.Fatalf("failed to create export directory %s: %v", dir, err)
	}

	e.run(targets)
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

// exporterImpl defines the interface for exporting parts of a Please repository to a new directory.
// It handles the copying of configuration files, preloaded build definitions, and selected
// targets along with their necessary source files and dependencies.
type exporterImpl interface {
	// exportPreloaded exports all globally preloaded build definitions and subincluded targets.
	// These are usually defined in the repository's configuration file.
	exportPreloaded()
	// exportTarget exports an individual build target.
	// Each target recursively exports all their source files and required build statements, but also
	// targets in their transitive dependencies.
	exportTarget(*core.BuildTarget)
	// writePackageFiles writes the processed BUILD files for all exported targets to the
	// export directory. These BUILD files may be modified (e.g., trimmed) depending on
	// the exporter's implementation.
	writePackageFiles()
}

// newExporter creates a new exporter of a specific type based on the arguments.
func newExporter(state *core.BuildState, dir string, noTrim bool) *baseExporter {
	base := &baseExporter{
		state:           state,
		targetDir:       dir,
		exportedTargets: map[core.BuildLabel]bool{},
	}

	var exporter exporterImpl
	if noTrim {
		exporter = newNoTrimExporter(base)
	} else {
		exporter = newTrimmedExporter(base)
	}

	base.impl = exporter
	return base
}

// baseExporter provides common fields and methods of other exporters.
type baseExporter struct {
	state         *core.BuildState
	targetDir     string
	targetCounter int

	// exportedTargets maintains a record of the targets that have been exported so far.
	exportedTargets map[core.BuildLabel]bool
	// impl is a reference to the concrete exporter implementation. It's included for calling the
	// specific exporter implementation from the common methods.
	impl exporterImpl
}

// run specifies the main steps when running an export.
func (be *baseExporter) run(targets core.BuildLabels) {
	go be.startMonitor()
	be.exportRepoConfig()
	be.impl.exportPreloaded()
	be.exportTargets(targets)
	be.impl.writePackageFiles()
}

func (be *baseExporter) startMonitor() {
	for {
		time.Sleep(10 * time.Second)
		log.Infof("Number of targets exported: %v", be.targetCounter)
	}
}

// exportRepoConfig exports the repository's configuration files (e.g., .gitignore, .plzconfig and its
// platform-specific variants) to the target export directory.
func (be *baseExporter) exportRepoConfig() {
	files, err := filepath.Glob(".plzconfig*")
	if err != nil {
		log.Fatalf("failed to glob .plzconfig files: %v", err)
	}
	if info, err := os.Stat(".gitignore"); err == nil {
		files = append(files, info.Name())
	}

	for _, file := range files {
		targetPath := filepath.Join(be.targetDir, file)
		if err := os.RemoveAll(targetPath); err != nil {
			log.Fatalf("failed to remove file %s: %v", file, err)
		}
		if err := fs.CopyFile(file, targetPath, 0); err != nil {
			log.Fatalf("failed to copy file %s: %v", file, err)
		}
	}
}

// exportTargets exports the set of targets identified by the given build labels.
func (be *baseExporter) exportTargets(labels core.BuildLabels) {
	for _, l := range labels {
		if be.exportedTargets[l] {
			continue
		}
		target := be.getOrParseTarget(l)
		if target == nil {
			log.Errorf("Unable to lookup target %s", l)
			continue
		}
		be.impl.exportTarget(target)
	}
}

// exportDependencies exports dependencies of a target.
func (be *baseExporter) exportDependencies(target *core.BuildTarget) {
	deps := target.DeclaredDependencies()
	log.Debugf("Exporting dependencies of (%v): %v", target.Label, deps)
	be.exportTargets(deps)
}

// exportSources exports all files required by the target.
func (be *baseExporter) exportSources(target *core.BuildTarget) {
	for _, src := range target.AllBuildInputs() {
		if _, ok := src.Label(); ok {
			continue // These will be handled as dependencies later
		}
		for _, p := range src.Paths(be.state.Graph) {
			if filepath.IsAbs(p) { // Don't copy system file deps.
				log.Debugf("System dependency detected, skipping...: %s", p)
				continue
			}
			dest := filepath.Join(be.targetDir, p)
			if target.Subrepo != nil { // Adjusting fo for local subrepos
				dest = filepath.Join(be.targetDir, target.Subrepo.Dir(p))
			}
			if err := fs.RecursiveCopy(p, dest, 0); err != nil {
				log.Warningf("Error copying file, skipping...: %s", err)
			}
			log.Debugf("Writing exported source file: %s", p)
		}
	}
}

// getOrParseTarget attempts to lookup a target in the build graph. If the target has not
// been parsed yet, it dynamically requests the package be parsed and blocks until the target is resolved.
//
// This occurs during the exports when walking dependencies of adjacent targets which were not
// explicitly activated or resolved during the initial build/parse phase.
//
// This requires the background parser worker threads to be kept alive as daemons (controlled by the
// "KeepParserRunning" build state option).
func (be *baseExporter) getOrParseTarget(label core.BuildLabel) *core.BuildTarget {
	target := be.state.Graph.Target(label)
	if target == nil {
		log.Infof("Target %v not found in graph. Attempting to parse...", label)
		parse.Parse(be.state, label, core.OriginalTarget, core.ParseModeNormal)
		target = be.state.Graph.Target(label)
	}
	return target
}

// getOrParsePackage attempts to lookup a package in the build graph. If the package has not
// been parsed yet, it dynamically requests the package be parsed and blocks until resolved.
//
// This requires the background parser worker threads to be kept alive as daemons (controlled by the
// "KeepParserRunning" build state option).
func (be *baseExporter) getOrParsePackage(label core.BuildLabel) *core.Package {
	return be.state.WaitForPackage(label, core.OriginalTarget, core.ParseModeNormal)
}

// checkAndSetVisited is a helper to ensure we only visit the same target once.
// It returns true if this is the first time the target is being exported.
func (be *baseExporter) checkAndSetVisited(target *core.BuildTarget) bool {
	visited := be.exportedTargets[target.Label]
	be.exportedTargets[target.Label] = true
	be.targetCounter++
	return !visited
}
