// Package export handles exporting parts of the repo to other directories.
// This is useful if, for example, one wanted to separate out part of
// their repo with all dependencies.
package export

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/please-build/buildtools/build"

	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/parse"
	"github.com/thought-machine/please/src/parse/asp"
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
	ExportTarget(target *core.BuildTarget)
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

// defaultExporter implements an exporter that trims packages to reach a minimal exported repo.
type defaultExporter struct {
	baseExporter
	// visitedPackages maintains a record of the packages visited during the export process.
	visitedPackages map[core.BuildLabel]bool
	// requiredSubincludes maps packages to the subinclude labels they require.
	requiredSubincludes map[core.BuildLabel]core.BuildLabels
	// preloadedSubincludes tracks subincludes that are preloaded and don't need explicit export.
	preloadedSubincludes map[core.BuildLabel]bool
}

func (e *defaultExporter) ExportPreloaded() {
	// Write any preloaded build defs
	for _, preload := range e.state.Config.Parse.PreloadBuildDefs {
		if err := fs.RecursiveCopy(preload, filepath.Join(e.targetDir, preload), 0); err != nil {
			log.Fatalf("Failed to copy preloaded build def %s: %s", preload, err)
		}
	}

	for _, target := range e.state.Config.Parse.PreloadSubincludes {
		targets := append(e.state.Graph.TransitiveSubincludes(target), target)
		for _, t := range targets {
			e.preloadedSubincludes[t] = true
		}
		e.ExportTargets(targets)
	}
}

func (e *defaultExporter) ExportTarget(target *core.BuildTarget) {
	if !e.checkAndSetVisited(target) {
		return
	}

	log.Debugf("Exporting target: %v", target.Label)

	// Skip export for internal packages
	if target.Label.PackageName == parse.InternalPackageName {
		return
	}
	// We want to export the package that made this subrepo available, but we still need to walk the
	// target deps as it may depend on other subrepos or first party targets
	if target.Subrepo != nil && target.Subrepo.Target != nil {
		e.ExportTarget(target.Subrepo.Target)
		e.exportDependencies(target)
		return
	}

	e.exportSources(target)
	e.exportDependencies(target)

	pkg := e.state.Graph.PackageOrDie(target.Label)
	e.exportSubincludes(pkg, target)
	e.exportRelatedTargets(pkg, target)
	e.visitedPackages[pkg.Label()] = true
}

func (e *defaultExporter) WritePackageFiles() {
	for pkgLabel := range e.visitedPackages {
		pkg := e.state.Graph.PackageOrDie(pkgLabel)
		filteredBytes, err := e.trimPackage(pkg)
		if err != nil {
			log.Errorf("Failed to filter the build statements of package %s: %v", pkg.Label(), err)
			continue
		}

		parsedBuild, err := build.ParseBuild(pkg.Filename, filteredBytes)
		if err != nil {
			log.Fatalf("Failed to parse bytes for formatting: %v\nData:\n%s", err, filteredBytes)
		}
		formattedBytes := build.Format(parsedBuild)

		e.WriteExportedPackageFile(pkg, formattedBytes)
	}
}

// exportSubincludes exports the subincluded targets required to generate the target and selects them to
// later be written to the package as statements.
func (e *defaultExporter) exportSubincludes(pkg *core.Package, target *core.BuildTarget) {
	for _, subinclude := range pkg.Metadata.FindRequiredSubincludes(target) {
		// skip for preloaded subincludes, these are handled separately at the start to ensure they are
		// they are exported even if not directly used by an exported target.
		if e.preloadedSubincludes[subinclude] {
			continue
		}

		required := e.requiredSubincludes[pkg.Label()]
		if !slices.Contains(required, subinclude) {
			required = append(required, subinclude)
		}
		e.requiredSubincludes[pkg.Label()] = required

		target := e.getOrParseTarget(subinclude)
		if target == nil {
			log.Errorf("Unable to lookup target %s", subinclude)
			continue
		}
		e.ExportTarget(target)
	}
}

// exportRelatedTargets exports build targets that are related to the build statement that generated.
func (e *defaultExporter) exportRelatedTargets(pkg *core.Package, target *core.BuildTarget) {
	stmt := pkg.Metadata.FindStatement(target)
	if stmt == nil {
		log.Errorf("Failed to find statement for target %s in %s", target, pkg.Name)
		return
	}

	relatedTargets := pkg.Metadata.FindTargets(stmt)
	log.Debugf("Exporting targets related to %s: %v", target, relatedTargets)
	for _, target := range relatedTargets {
		e.ExportTarget(target)
	}
}

// WriteExportedPackageFile creates a new package (BUILD) file in the exported dir and writes to it.
func (e *defaultExporter) WriteExportedPackageFile(pkg *core.Package, content []byte) {
	filename := pkg.Filename
	exportedFilename := filepath.Join(e.targetDir, filename)
	f, err := fs.OpenDirFile(exportedFilename, os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		log.Fatalf("Failed to create and open exported BUILD file for %s: %v", exportedFilename, err)
	}
	defer f.Close()

	if _, err := f.Write(content); err != nil {
		log.Errorf("Failed to write to exported BUILD file %s: %v", f.Name(), err)
	}
}

// trimPackage filters the statements to be written to the exported BUILD file.
func (e *defaultExporter) trimPackage(pkg *core.Package) ([]byte, error) {
	p := asp.NewParserOnly()
	parsed, err := p.ParseFileOnly(pkg.Filename)
	if err != nil {
		return nil, fmt.Errorf("Parsing original BUILD file: %v", err)
	}

	content, err := os.ReadFile(pkg.Filename)
	if err != nil {
		return nil, fmt.Errorf("Opening original BUILD file: %v", err)
	}

	trimmer := trimmer{
		origin:   content,
		pkg:      pkg,
		exporter: e,
		// assuming max len of the original file to avoid reallocations.
		bytes: make([]byte, 0, len(content)),
	}
	trimmer.trimBlock(parsed, 0, asp.Position(len(content)))

	return trimmer.bytes, nil
}

// noTrimExporter implements an exporter that avoids trimming any packages by exporting all targets
// and statements in a package.
type noTrimExporter struct {
	baseExporter
	// exportedPackages tracks which packages have already had their BUILD files exported.
	exportedPackages map[string]bool
}

func (nte *noTrimExporter) ExportPreloaded() {
	// Write any preloaded build defs
	for _, preload := range nte.state.Config.Parse.PreloadBuildDefs {
		if err := fs.RecursiveCopy(preload, filepath.Join(nte.targetDir, preload), 0); err != nil {
			log.Errorf("Failed to copy preloaded build def %s: %s", preload, err)
		}
	}

	for _, target := range nte.state.Config.Parse.PreloadSubincludes {
		targets := append(nte.state.Graph.TransitiveSubincludes(target), target)
		nte.ExportTargets(targets)
	}
}

func (nte *noTrimExporter) ExportTarget(target *core.BuildTarget) {
	pkg := nte.state.Graph.PackageOrDie(target.Label)
	if !nte.checkAndSetVisited(target) {
		return
	}

	// We want to export the package that made this subrepo available, but we still need to walk the target deps
	// as it may depend on other subrepos or first party targets
	if target.Subrepo != nil {
		nte.ExportTarget(target.Subrepo.Target)
		nte.exportDependencies(target)
		return
	}

	nte.exportPackage(pkg)
	nte.exportSubincludes(pkg)
	nte.exportAllTargets(pkg)
	nte.exportSources(target)
	nte.exportDependencies(target)
}

func (nte *noTrimExporter) WritePackageFiles() {
}

// exportPackage exports the package BUILD file.
func (nte *noTrimExporter) exportPackage(pkg *core.Package) {
	// Skip subrepos and internal packages. These will be generated by build statements in the exported
	// repo or included in please internally.
	if pkg.Subrepo != nil || pkg.Name == parse.InternalPackageName {
		return
	}

	if nte.exportedPackages[pkg.Name] {
		return
	}
	nte.exportedPackages[pkg.Name] = true

	exportedFilename := filepath.Join(nte.targetDir, pkg.Filename)
	if err := fs.CopyFile(pkg.Filename, exportedFilename, 0); err != nil {
		log.Errorf("failed to export package %s: %v", pkg.Name, err)
	}
}

// exportSubincludes exports the subincluded targets.
func (nte *noTrimExporter) exportSubincludes(pkg *core.Package) {
	subincludes := pkg.AllSubincludes(nte.state.Graph)
	nte.ExportTargets(subincludes)
}

// exportAllTargets will export all the targets in the provided package.
func (nte *noTrimExporter) exportAllTargets(pkg *core.Package) {
	for _, target := range pkg.AllTargets() {
		nte.ExportTarget(target)
	}
}
