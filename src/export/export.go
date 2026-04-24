// Package export handles exporting parts of the repo to other directories.
// This is useful if, for example, one wanted to separate out part of
// their repo with all dependencies.
package export

import (
	"bytes"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"

	"github.com/please-build/buildtools/build"

	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/parse"
	"github.com/thought-machine/please/src/parse/asp"
)

var log = logging.Log

type Exporter interface {
	PlzConfig()
	Preloaded()
	Targets(core.BuildLabels)
	Target(target *core.BuildTarget)
	WritePackageFiles()
}

func Repo(state *core.BuildState, dir string, noTrim bool, targets []core.BuildLabel) {
	e := newExporter(state, dir, noTrim)

	// ensure output dir
	if err := os.MkdirAll(dir, fs.DirPermissions); err != nil {
		log.Fatalf("failed to create export directory %s: %v", dir, err)
	}

	e.PlzConfig()
	e.Preloaded()
	e.Targets(targets)
	e.WritePackageFiles()
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

func newExporter(state *core.BuildState, dir string, noTrim bool) Exporter {
	base := baseExporter{
		state:           state,
		targetDir:       dir,
		exportedTargets: map[*core.Package]map[core.BuildLabel]bool{},
	}

	if noTrim {
		exporter := &NoTrimExporter{
			baseExporter:     base,
			exportedPackages: map[string]bool{},
		}
		exporter.impl = exporter
		return exporter
	} else {
		exporter := &DefaultExporter{
			baseExporter:         base,
			requiredSubincludes:  map[*core.Package]map[core.BuildLabel]bool{},
			preloadedSubincludes: map[core.BuildLabel]bool{},
		}
		exporter.impl = exporter
		return exporter
	}
}

// baseExporter provides common fields and methods of other exporters. A reference
// to the concrete exporter implementation is included to be used in the common methods.
type baseExporter struct {
	state     *core.BuildState
	targetDir string

	exportedTargets map[*core.Package]map[core.BuildLabel]bool
	impl            Exporter
}

// PlzConfig exports the repo configuration files.
func (be *baseExporter) PlzConfig() {
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

// Targets exports all targets for the given labels.
func (be *baseExporter) Targets(labels core.BuildLabels) {
	for _, l := range labels {
		target := be.state.Graph.TargetOrDie(l)
		be.impl.Target(target)
	}
}

// Dependencies exports dependencies of a target.
func (be *baseExporter) Dependencies(target *core.BuildTarget) {
	for _, dep := range target.Dependencies() {
		be.impl.Target(dep)
	}
}

// Sources exports all files required by the target.
func (be *baseExporter) Sources(target *core.BuildTarget) {
	for _, src := range append(target.AllSources(), target.AllData()...) {
		if _, ok := src.Label(); !ok { // We'll handle these dependencies later
			for _, p := range src.Paths(be.state.Graph) {
				if !filepath.IsAbs(p) { // Don't copy system file deps.
					if err := fs.RecursiveCopy(p, filepath.Join(be.targetDir, p), 0); err != nil {
						log.Fatalf("Error copying file: %s\n", err)
					}
					log.Warning("Writing source file: %s", p)
				}
			}
		}
	}
}

func (be *baseExporter) checkFirstExport(pkg *core.Package, target *core.BuildTarget) bool {
	if _, ok := be.exportedTargets[pkg]; !ok {
		be.exportedTargets[pkg] = map[core.BuildLabel]bool{}
	}
	if be.exportedTargets[pkg][target.Label] {
		return false
	}
	be.exportedTargets[pkg][target.Label] = true
	return true
}

// DefaultExporter implements an exporter that trims packages to reach a minimal exported repo.
type DefaultExporter struct {
	baseExporter
	requiredSubincludes  map[*core.Package]map[core.BuildLabel]bool
	preloadedSubincludes map[core.BuildLabel]bool
}

// Preloaded exports the preloaded targets, build defs and subincludes. These preloads are usually
// defined in the .plzexport config.
func (e *DefaultExporter) Preloaded() {
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
		e.Targets(targets)
	}
}

// Target exports an individual target. This implementation will attempt to export a minimal repo
// with only the required targets and statements.
func (e *DefaultExporter) Target(target *core.BuildTarget) {
	pkg := e.state.Graph.PackageOrDie(target.Label)
	if e.checkFirstExport(pkg, target) == false {
		return
	}

	// We want to export the package that made this subrepo available, but we still need to walk the target deps
	// as it may depend on other subrepos or first party targets
	if target.Subrepo != nil {
		e.Target(target.Subrepo.Target)
		// TODO do we need dependencies and sources?
		return
	}

	e.Subincludes(pkg, target)
	e.BuildStatements(pkg, target)
	e.Sources(target)
	e.Dependencies(target)
}

// Subincludes exports the subincluded targets required to generate the target and selects them to
// later be written to the package as statements.
func (e *DefaultExporter) Subincludes(pkg *core.Package, target *core.BuildTarget) {
	subincludes, err := pkg.FindRequiredSubincludes(target)
	if err != nil {
		log.Infof("No subincludes found, assuming non required.: %w", pkg.Name, err)
		return
	}

	for _, subinclude := range subincludes {
		// skip for preloaded subincludes
		if e.preloadedSubincludes[subinclude] {
			continue
		}

		if _, ok := e.requiredSubincludes[pkg]; !ok {
			e.requiredSubincludes[pkg] = map[core.BuildLabel]bool{}
		}
		e.requiredSubincludes[pkg][subinclude] = true

		e.Target(e.state.Graph.TargetOrDie(subinclude))
	}

	log.Warningf("Parse Metadata Subincludes: %v", pkg.BuildFileMetadata.TargetToSubinclude)
}

// BuildStatements exports BUILD statements that generate the build target.
func (e *DefaultExporter) BuildStatements(pkg *core.Package, target *core.BuildTarget) {
	if target.Label.PackageName == parse.InternalPackageName {
		// TODO validate if we still need this
		return
	}

	stmt, err := pkg.FindStatement(target)
	if err != nil {
		log.Fatalf("Failed to find statement in %s: %w", pkg.Name, err)
	}

	relatedTargets, err := pkg.FindRelatedTargets(stmt)
	if err != nil {
		log.Fatalf("Failed to lookup related targets for package %s: %w", pkg.Name, err)
	}

	for _, target := range relatedTargets {
		e.Target(target)
	}
}

// WritePackageFiles writes the trimmed BUILD files to the export directory.
func (e *DefaultExporter) WritePackageFiles() {
	for pkg, labels := range e.exportedTargets {
		log.Warningf("On package %v Selected targets: %v", pkg, slices.Collect(maps.Keys(labels)))

		// filter
		filteredBytes, err := e.FilterPackageFile(pkg)
		if err != nil {
			log.Fatalf("Failed to filter the build statements of package %s: %v", pkg.Label(), err)
		}

		// format
		buildParser, err := build.ParseBuild(pkg.Filename, filteredBytes)
		formatedBytes := build.Format(buildParser)

		// write
		file := e.OpenExportedPackageFile(pkg)
		defer file.Close()
		if _, err := file.Write(formatedBytes); err != nil {
			log.Fatalf("Failed to write to exported BUILD file %s: %v", file.Name(), err)
		}
	}
}

func (e *DefaultExporter) OpenExportedPackageFile(pkg *core.Package) *os.File {
	filename := pkg.Filename
	exportedFilename := filepath.Join(e.targetDir, filename)
	f, err := fs.OpenDirFile(exportedFilename, os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		log.Fatalf("failed to create and open exported BUILD file for %s: %v", exportedFilename, err)
	}
	return f
}

// FilterPackageFile filters the statements to be written to the exported BUILD file.
func (e *DefaultExporter) FilterPackageFile(pkg *core.Package) ([]byte, error) {
	p := asp.NewParserOnly()
	parsedStmts, err := p.ParseFileOnly(pkg.Filename)
	if err != nil {
		log.Fatalf("failed to parse original BUILD file: %v", err)
	}

	original, err := os.ReadFile(pkg.Filename)
	if err != nil {
		log.Fatalf("failed to open original BUILD file: %v", err)
	}

	cursor := 0
	var buffer bytes.Buffer
	for _, stmt := range parsedStmts {
		bStmt := asp.NewBuildStatement(stmt)

		if cursor < bStmt.Start {
			if _, err := buffer.Write(original[cursor:bStmt.Start]); err != nil {
				return nil, err
			}
			cursor = bStmt.Start
		}

		// Write filtered subincludes
		if stmtLabels, ok := pkg.GetSubincludedLabels(bStmt); ok {

			subStmt := e.makeSubincludeStatement(stmtLabels, e.requiredSubincludes[pkg])
			buffer.Write([]byte(subStmt))
			cursor = bStmt.End
			continue
		}

		// Don't write statements that generate targets we are not interested about
		if targets, err := pkg.FindRelatedTargets(bStmt); err == nil {
			needed := false
			for _, target := range targets {
				if e.exportedTargets[pkg][target.Label] {
					needed = true
				}
			}
			if needed == false {
				// don't write
				cursor = bStmt.End
				continue
			}
		}

		// Write every other statement
		if buffer.Write(original[bStmt.Start:bStmt.End]); err != nil {
			return nil, err
		}
		cursor = bStmt.End
	}

	// Write the rest of the original file (non build targets)
	if buffer.Write(original[cursor:]); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// makeSubincludeStatement generates a single subinclude statement (as string) with the argument labels.
func (e *DefaultExporter) makeSubincludeStatement(available core.BuildLabels, required map[core.BuildLabel]bool) string {
	// make subincludes that contains a subset of the labels defined in the statement
	var filteredLabels core.BuildLabels
	for _, label := range available {
		if required[label] {
			filteredLabels = append(filteredLabels, label)
		}
	}
	sort.Sort(filteredLabels)

	call := &build.CallExpr{
		X: &build.Ident{Name: "subinclude"},
	}
	for _, label := range filteredLabels {
		call.List = append(call.List, &build.StringExpr{Value: label.String()})
	}

	return build.FormatString(call)
}

// NoTrimExporter implements an exporter that avoids trimming any packages by exporting all targets
// and statements in a package.
type NoTrimExporter struct {
	baseExporter
	exportedPackages map[string]bool
}

func (nte *NoTrimExporter) Preloaded() {
	// Write any preloaded build defs
	for _, preload := range nte.state.Config.Parse.PreloadBuildDefs {
		if err := fs.RecursiveCopy(preload, filepath.Join(nte.targetDir, preload), 0); err != nil {
			log.Fatalf("Failed to copy preloaded build def %s: %s", preload, err)
		}
	}

	for _, target := range nte.state.Config.Parse.PreloadSubincludes {
		targets := append(nte.state.Graph.TransitiveSubincludes(target), target)
		nte.Targets(targets)
	}
}

// Target exports an individual target. This implementation won't attempted any trimming, exporting
// all targets and statements defined in the package.
func (nte *NoTrimExporter) Target(target *core.BuildTarget) {
	pkg := nte.state.Graph.PackageOrDie(target.Label)
	if nte.checkFirstExport(pkg, target) == false {
		return
	}

	// We want to export the package that made this subrepo available, but we still need to walk the target deps
	// as it may depend on other subrepos or first party targets
	if target.Subrepo != nil {
		nte.Target(target.Subrepo.Target)
		// TODO do we need dependencies and sources?
		return
	}

	nte.Package(pkg)
	nte.Subincludes(pkg, target)
	nte.AllTargets(pkg)
	nte.Sources(target)
	nte.Dependencies(target)
}

// Package exports the package BUILD file.
func (nte *NoTrimExporter) Package(pkg *core.Package) {
	pkgName := pkg.Name
	if pkgName == parse.InternalPackageName {
		return
	}
	if nte.exportedPackages[pkgName] {
		return
	}
	nte.exportedPackages[pkgName] = true

	pkgFilename := pkg.Filename
	exportedFilename := filepath.Join(nte.targetDir, pkgFilename)

	if err := fs.CopyFile(pkgFilename, exportedFilename, 0); err != nil {
		log.Fatalf("failed to export package %s: %v", pkgName, err)
	}
}

// Subincludes exports the subincluded targets.
func (nte *NoTrimExporter) Subincludes(pkg *core.Package, target *core.BuildTarget) {
	subincludes := pkg.AllSubincludes(nte.state.Graph)
	for _, subinclude := range subincludes {
		nte.Target(nte.state.Graph.TargetOrDie(subinclude))
	}
}

// AllTargets will export all the targets in the provided package.
func (nte *NoTrimExporter) AllTargets(pkg *core.Package) {
	for _, target := range pkg.AllTargets() {
		nte.Target(target)
	}
}

// WritePackageFiles in the NoTrimExporter doesn't require an implementation due to total copy
// of BUILD file.
func (nte *NoTrimExporter) WritePackageFiles() {
	return
}
