// Package export handles exporting parts of the repo to other directories.
// This is useful if, for example, one wanted to separate out part of
// their repo with all dependencies.
package export

import (
	"bufio"
	"io"
	iofs "io/fs"
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
)

var log = logging.Log

type Exporter interface {
	PlzConfig()
	Preloaded()
	Targets(core.BuildLabels)
	Target(target *core.BuildTarget)
	WriteBuildStatements()
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
	e.WriteBuildStatements()
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
		exportedTargets: map[core.BuildLabel]bool{},
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
			selectedStatements:   map[*core.Package]map[core.BuildStatement]bool{},
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

	exportedTargets map[core.BuildLabel]bool
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

// DefaultExporter implements an exporter that trims packages to reach a minimal exported repo.
type DefaultExporter struct {
	baseExporter
	selectedStatements   map[*core.Package]map[core.BuildStatement]bool
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
	if e.exportedTargets[target.Label] {
		return
	}
	e.exportedTargets[target.Label] = true

	// We want to export the package that made this subrepo available, but we still need to walk the target deps
	// as it may depend on other subrepos or first party targets
	if target.Subrepo != nil {
		e.Target(target.Subrepo.Target)
		// TODO do we need dependencies and sources?
		return
	}

	pkg := e.state.Graph.PackageOrDie(target.Label)

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

	if _, ok := e.selectedStatements[pkg]; !ok {
		e.selectedStatements[pkg] = map[core.BuildStatement]bool{}
	}

	stmt, err := pkg.FindStatement(target)
	if err != nil {
		log.Fatalf("Failed to find statement in %s: %w", pkg.Name, err)
	}

	// check if visited before
	if e.selectedStatements[pkg][*stmt] == true {
		return
	}
	e.selectedStatements[pkg][*stmt] = true

	relatedTargets, err := pkg.FindRelatedTargets(stmt)
	if err != nil {
		log.Fatalf("Failed to lookup related targets for package %s: %w", pkg.Name, err)
	}

	for _, target := range relatedTargets {
		e.Target(target)
	}
}

// WriteBuildStatements writes the BUILD file statements to the export directory.
func (e *DefaultExporter) WriteBuildStatements() {
	log.Warningf("Selected Statements: %v", e.selectedStatements)

	for pkg, stmtMap := range e.selectedStatements {
		stmts := slices.Collect(maps.Keys(stmtMap))
		// Sort statements by position to keep them in order
		sort.Sort(core.BuildStatements(stmts))

		e.writePackageFile(pkg, stmts)
	}
}

// writePackageFile writes the selected statements to the please build file in the exported directory.
func (e *DefaultExporter) writePackageFile(pkg *core.Package, stmts []core.BuildStatement) {
	filename := pkg.Filename
	exportedFilename := filepath.Join(e.targetDir, filename)

	log.Warningf("Writing file: %s", filename)

	fr, err := os.Open(filename)
	if err != nil {
		log.Fatalf("failed to open file original BUILD file: %v", err)
		return
	}
	defer fr.Close()

	frStat, err := fr.Stat()
	if err != nil {
		log.Fatalf("failed to get original BUILD file status: %v", err)
	}

	fw, err := fs.OpenDirFile(exportedFilename, os.O_CREATE|os.O_WRONLY, frStat.Mode())
	if err != nil {
		log.Fatalf("failed to create and open exported BUILD file for %s: %v", exportedFilename, err)
	}
	defer fw.Close()

	writer := bufio.NewWriter(fw)
	// Subinclude
	if subinclude := e.makeSubincludesStatement(pkg); subinclude != "" {
		if _, err := writer.WriteString(subinclude + "\n\n"); err != nil {
			log.Fatalf("failed to add subincludes to %s: %v", exportedFilename, err)
		}
	}
	// Statements
	for _, s := range stmts {
		if _, err := fr.Seek(s.StartPos(), io.SeekStart); err != nil {
			log.Fatalf("failed to seek in BUILD file %s: %v", filename, err)
		}

		if _, err := io.CopyN(writer, fr, s.Len()); err != nil {
			log.Fatalf("failed to copy statement from %s to %s: %v", filename, exportedFilename, err)
		}

		if _, err := writer.WriteString("\n"); err != nil {
			log.Fatalf("failed to add newline to %s: %v", exportedFilename, err)
		}
	}
	if err := writer.Flush(); err != nil {
		log.Fatalf("failed write exported BUILD file %s: %v", exportedFilename, err)
	}
}

// makeSubincludesStatement generates a single subinclude statement (as string) with the required
// targets for this package.
func (e *DefaultExporter) makeSubincludesStatement(pkg *core.Package) string {
	subincludesMap, ok := e.requiredSubincludes[pkg]
	if !ok || len(subincludesMap) == 0 {
		return ""
	}

	labels := slices.Collect(maps.Keys(subincludesMap))
	sort.Sort(core.BuildLabels(labels))

	call := &build.CallExpr{
		X: &build.Ident{Name: "subinclude"},
	}
	for _, label := range labels {
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
	if nte.exportedTargets[target.Label] {
		return
	}
	nte.exportedTargets[target.Label] = true

	// We want to export the package that made this subrepo available, but we still need to walk the target deps
	// as it may depend on other subrepos or first party targets
	if target.Subrepo != nil {
		nte.Target(target.Subrepo.Target)
		// TODO do we need dependencies and sources?
		return
	}

	pkg := nte.state.Graph.PackageOrDie(target.Label)

	nte.Package(target)
	nte.Subincludes(pkg, target)
	nte.AllTargets(pkg)
	nte.Dependencies(target)
}

var ignoreDirectories = map[string]bool{
	"plz-out": true,
	".git":    true,
	".svn":    true,
	".hg":     true,
}

// Package exports the package BUILD file containing the given target and all sources.
func (nte *NoTrimExporter) Package(target *core.BuildTarget) {
	pkgName := target.Label.PackageName
	if pkgName == parse.InternalPackageName {
		return
	}
	if nte.exportedPackages[pkgName] {
		return
	}
	nte.exportedPackages[pkgName] = true

	pkgDir := filepath.Clean(pkgName)

	err := filepath.WalkDir(pkgDir, func(path string, d iofs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != pkgDir && fs.IsPackage(nte.state.Config.Parse.BuildFileName, path) {
				return filepath.SkipDir // We want to stop when we find another package in our dir tree
			}
			if ignoreDirectories[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil // Ignore symlinks, which are almost certainly generated sources.
		}
		dest := filepath.Join(nte.targetDir, path)
		if err := fs.EnsureDir(dest); err != nil {
			return err
		}
		return fs.CopyFile(path, dest, 0)
	})
	if err != nil {
		log.Fatalf("failed to export package %s for %s: %v", pkgName, target.Label, err)
	}
}

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

// WriteBuildStatements in the NoTrimExporter doesn't require an implementation due to total copy
// of BUILD package.
func (nte *NoTrimExporter) WriteBuildStatements() {
	return
}
