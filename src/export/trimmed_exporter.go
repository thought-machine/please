package export

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/parse"
	"github.com/thought-machine/please/src/parse/asp"
)

// trimmedExporter implements an exporter that trims packages to reach a minimal exported repo.
type trimmedExporter struct {
	*baseExporter
	// visitedPackages maintains a record of the packages visited during the export process.
	visitedPackages map[core.BuildLabel]bool
	// requiredSubincludes maps packages to the subinclude labels they require.
	requiredSubincludes map[core.BuildLabel]core.BuildLabels
	// preloadedSubincludes tracks subincludes that are preloaded and don't need explicit export.
	preloadedSubincludes map[core.BuildLabel]bool
}

func newTrimmedExporter(base *baseExporter) exporterImpl {
	return &trimmedExporter{
		baseExporter:         base,
		visitedPackages:      map[core.BuildLabel]bool{},
		requiredSubincludes:  map[core.BuildLabel]core.BuildLabels{},
		preloadedSubincludes: map[core.BuildLabel]bool{},
	}
}

// exportPreloaded implements [exporterImpl].
func (e *trimmedExporter) exportPreloaded() {
	// Write any preloaded build defs
	for _, preload := range e.state.Config.Parse.PreloadBuildDefs {
		if err := fs.RecursiveCopy(preload, filepath.Join(e.targetDir, preload), 0); err != nil {
			log.Fatalf("Failed to copy preloaded build def %s: %s", preload, err)
		}
	}

	for _, target := range e.state.GetPreloadedSubincludes() {
		targets := append(e.state.Graph.TransitiveSubincludes(target), target)
		for _, t := range targets {
			e.preloadedSubincludes[t] = true
		}
		e.exportTargets(targets)
	}
}

// exportTarget implements [exporterImpl].
func (e *trimmedExporter) exportTarget(target *core.BuildTarget) {
	if !e.checkAndSetVisited(target) {
		return
	}

	// Skip export for internal packages
	if target.Label.PackageName == parse.InternalPackageName {
		return
	}
	// We want to export the package that made this subrepo available, but we still need to walk the
	// target deps as it may depend on other subrepos or first party targets
	if target.Subrepo.IsExternal() {
		e.exportTarget(target.Subrepo.Target)
		e.exportDependencies(target)
		return
	}

	e.exportSources(target)
	e.exportDependencies(target)

	pkg := e.getOrParsePackage(target.Label)
	if pkg == nil {
		log.Errorf("Unable to lookup package %s", target.Label)
		return
	}
	e.exportSubincludes(pkg, target.Label)
	e.exportRelatedTargets(pkg, target.Label)

	if !e.visitedPackages[pkg.Label()] {
		// Export subincluded targets required for other package statements, e.g. variable
		// declaration, during the first visit of a package.
		e.exportPackageRequirements(pkg)
		e.visitedPackages[pkg.Label()] = true
	}
}

// writePackageFiles implements [exporterImpl].
func (e *trimmedExporter) writePackageFiles() {
	p := asp.NewParserOnly()
	for pkgLabel := range e.visitedPackages {
		pkg := e.state.Graph.PackageOrDie(pkgLabel)
		filteredBytes, err := e.trimPackage(p, pkg)
		if err != nil {
			log.Errorf("Failed to filter the build statements of package %s: %v", pkg.Label(), err)
			continue
		}

		e.writeExportedPackageFile(pkg, trimNewlines(filteredBytes))
	}
}

// exportSubincludes exports the subincluded targets required to generate the target and selects them to
// later be written to the package as statements.
func (e *trimmedExporter) exportSubincludes(pkg *core.Package, target core.BuildLabel) {
	// Get the actively used subincludes of the target and propagate all transitive subincludes required
	// by our used subinclude targets. FindRequiredSubincludes will report the required subincludes
	// for this target at the package level but we need to propagate the subincluded targets inside
	// build definitions since we are not trimming build_defs files.
	usedSubincludes, err := pkg.Metadata.FindRequiredSubincludes(target)
	if err != nil {
		log.Fatalf("failed to find required subincludes for target %s: %s", target, err)
	}
	e.setPackageSubincludes(pkg, usedSubincludes)

	allSubincludes := usedSubincludes
	for _, sub := range usedSubincludes {
		for _, trans := range e.state.Graph.TransitiveSubincludes(sub) {
			if !slices.Contains(allSubincludes, trans) {
				allSubincludes = append(allSubincludes, trans)
			}
		}
	}

	e.exportTargets(allSubincludes)
}

// exportPackageRequirements exports any extra package requirements, for example the subincluded
// targets and files that are required by package but are not linked to any [core.BuildTarget].
func (e *trimmedExporter) exportPackageRequirements(pkg *core.Package) {
	subincludes, files := pkg.Metadata.FindPackageFileRequirements()
	e.setPackageSubincludes(pkg, subincludes)
	e.exportTargets(subincludes)
	e.exportFiles(files)
}

// setPackageSubincludes marks the package-level required subincludes after the export. This will be
// used for trimming subinclude statements with [trimmer].
func (e *trimmedExporter) setPackageSubincludes(pkg *core.Package, subincludes core.BuildLabels) {
	for _, subinclude := range subincludes {
		// skip for preloaded subincludes, these are handled separately at the start to ensure they are
		// exported even if not directly used by an exported target.
		if e.preloadedSubincludes[subinclude] {
			continue
		}

		pkgLabel := pkg.Label()
		required := e.requiredSubincludes[pkgLabel]
		if !slices.Contains(required, subinclude) {
			required = append(required, subinclude)
		}
		e.requiredSubincludes[pkgLabel] = required
	}
}

// exportRelatedTargets looks up and exports all build targets that were declared within the same
// build statement (e.g., adjacent targets in build def) as the specified target. This ensures that
// all co-defined targets are preserved in the exported BUILD file, preventing unresolved references
// or partial declarations.
func (e *trimmedExporter) exportRelatedTargets(pkg *core.Package, target core.BuildLabel) {
	relatedTargets, err := pkg.Metadata.FindRelatedTargets(target)
	if err != nil {
		log.Fatalf("failed to find related targets for %s: %s", target, err)
	}
	e.exportTargets(relatedTargets)
}

// WriteExportedPackageFile creates a new package (BUILD) file in the exported dir and writes to it.
func (e *trimmedExporter) writeExportedPackageFile(pkg *core.Package, content []byte) {
	filename := pkg.Filename
	exportedFilename := filepath.Join(e.targetDir, filename)
	if pkg.Subrepo != nil { // Adjusting fo for local subrepos
		exportedFilename = filepath.Join(e.targetDir, pkg.Subrepo.Dir(filename))
	}
	f, err := fs.OpenDirFile(exportedFilename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0664)
	if err != nil {
		log.Fatalf("Failed to create and open exported BUILD file for %s: %v", exportedFilename, err)
	}
	defer f.Close()

	if _, err := f.Write(content); err != nil {
		log.Errorf("Failed to write to exported BUILD file %s: %v", f.Name(), err)
	}
}

// trimPackage filters the statements to be written to the exported BUILD file.
func (e *trimmedExporter) trimPackage(p *asp.Parser, pkg *core.Package) ([]byte, error) {
	filename := pkg.Filename
	if pkg.Subrepo != nil { // Adjusting fo for local subrepos
		filename = pkg.Subrepo.Dir(filename)
	}

	parsed, err := p.ParseFileOnly(filename)
	if err != nil {
		return nil, fmt.Errorf("Parsing original BUILD file: %v", err)
	}

	content, err := os.ReadFile(filename)
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

// trimNewlines trims leading and trailing whitespace and collapses 3+ consecutive newlines to 2.
func trimNewlines(b []byte) []byte {
	trimmed := bytes.TrimSpace(b)
	var pointer, newlines int
	for _, val := range trimmed {
		if val == '\n' {
			newlines++
			if newlines > 2 {
				continue // Skip third (or more) consecutive newline
			}
		} else {
			newlines = 0
		}
		trimmed[pointer] = val
		pointer++
	}
	trimmed = trimmed[:pointer]

	if len(trimmed) > 0 {
		trimmed = append(trimmed, '\n') // Trailing newline
	}
	return trimmed
}
