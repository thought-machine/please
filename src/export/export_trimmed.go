package export

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/please-build/buildtools/build"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/parse"
	"github.com/thought-machine/please/src/parse/asp"
)

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
	subincludes := pkg.Metadata.FindRequiredSubincludes(target)
	for _, subinclude := range subincludes {
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
	}
	e.ExportTargets(subincludes)
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
