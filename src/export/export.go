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

type export struct {
	state     *core.BuildState
	targetDir string
	noTrim    bool

	exportedTargets    map[core.BuildLabel]bool
	exportedPackages   map[string]bool
	selectedStatements map[*core.Package]map[core.BuildStatement]bool
	requiredSubincludes map[*core.Package]map[core.BuildLabel]bool
	preloadedSubincludes map[core.BuildLabel]bool
}

func Repo(state *core.BuildState, dir string, noTrim bool, targets []core.BuildLabel) {
	e := newExport(state, dir, noTrim)

	// ensure output dir
	if err := os.MkdirAll(dir, fs.DirPermissions); err != nil {
		log.Fatalf("failed to create export directory %s: %v", dir, err)
	}

	e.plzConfig()
	e.preloaded()
	e.targets(targets)
	e.writeBuildStatements()
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

func newExport(state *core.BuildState, dir string, noTrim bool) *export {
	return &export{
		state:              state,
		noTrim:             noTrim,
		targetDir:          dir,
		exportedPackages:   map[string]bool{},
		exportedTargets:    map[core.BuildLabel]bool{},
		selectedStatements: map[*core.Package]map[core.BuildStatement]bool{},
		requiredSubincludes: map[*core.Package]map[core.BuildLabel]bool{},
		preloadedSubincludes: map[core.BuildLabel]bool{},
	}
}

func (e *export) plzConfig() {
	profiles, err := filepath.Glob(".plzconfig*")
	if err != nil {
		log.Fatalf("failed to glob .plzconfig files: %v", err)
	}
	for _, file := range profiles {
		targetPath := filepath.Join(e.targetDir, file)
		if err := os.RemoveAll(targetPath); err != nil {
			log.Fatalf("failed to remove .plzconfig file %s: %v", file, err)
		}
		if err := fs.CopyFile(file, targetPath, 0); err != nil {
			log.Fatalf("failed to copy .plzconfig file %s: %v", file, err)
		}
	}
}

func (e *export) preloaded() {
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
		e.targets(targets)
	}
}

func (e *export) targets(targets []core.BuildLabel) {
	for _, label := range targets {
		target := e.state.Graph.TargetOrDie(label)
		e.target(target)
	}
}

func (e *export) target(target *core.BuildTarget) {
	if e.exportedTargets[target.Label] {
		return
	}
	e.exportedTargets[target.Label] = true

	// We want to export the package that made this subrepo available, but we still need to walk the target deps
	// as it may depend on other subrepos or first party targets
	if target.Subrepo != nil {
		e.target(target.Subrepo.Target)
		// TODO do we need dependencies and sources?
		return
	}

	pkg := e.state.Graph.PackageOrDie(target.Label)

	// TODO notrim

	e.subincludes(pkg, target)
	e.buildStatements(pkg, target)
	e.sources(target)
	e.dependencies(target)
}

func (e *export) subincludes(pkg *core.Package, target *core.BuildTarget) {
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

		e.target(e.state.Graph.TargetOrDie(subinclude))
	}

	log.Warningf("Parse Metadata Subincludes: %v", pkg.BuildFileMetadata.TargetToSubinclude)
}

// buildStatements exports BUILD statements that generate the build target.
func (e *export) buildStatements(pkg *core.Package, target *core.BuildTarget) {
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
		e.target(target)
	}
}

func (e *export) sources(target *core.BuildTarget) {
	for _, src := range append(target.AllSources(), target.AllData()...) {
		if _, ok := src.Label(); !ok { // We'll handle these dependencies later
			for _, p := range src.FullPaths(e.state.Graph) {
				if !filepath.IsAbs(p) { // Don't copy system file deps.
					if err := fs.RecursiveCopy(p, filepath.Join(e.targetDir, p), 0); err != nil {
						log.Fatalf("Error copying file: %s\n", err)
					}
					log.Warning("Writing source file: %s", p)
				}
			}
		}
	}
}

func (e *export) dependencies(target *core.BuildTarget) {
	for _, dep := range target.Dependencies() {
		e.target(dep)
	}
}

var ignoreDirectories = map[string]bool{
	"plz-out": true,
	".git":    true,
	".svn":    true,
	".hg":     true,
}

// exportEntirePackage exports the package BUILD file containing the given target and all sources
func (e *export) exportEntirePackage(target *core.BuildTarget) {
	pkgName := target.Label.PackageName
	if pkgName == parse.InternalPackageName {
		return
	}
	if e.exportedPackages[pkgName] {
		return
	}
	e.exportedPackages[pkgName] = true

	pkgDir := filepath.Clean(pkgName)

	err := filepath.WalkDir(pkgDir, func(path string, d iofs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != pkgDir && fs.IsPackage(e.state.Config.Parse.BuildFileName, path) {
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
		dest := filepath.Join(e.targetDir, path)
		if err := fs.EnsureDir(dest); err != nil {
			return err
		}
		return fs.CopyFile(path, dest, 0)
	})
	if err != nil {
		log.Fatalf("failed to export package %s for %s: %v", pkgName, target.Label, err)
	}
}

// writeBuildStatements writes the BUILD file statements to the export directory.
func (e *export) writeBuildStatements() {
	log.Warningf("Selected Statements: %v", e.selectedStatements)

	for pkg, stmtMap := range e.selectedStatements {
		stmts := slices.Collect(maps.Keys(stmtMap))
		// Sort statements by position to keep them in order
		sort.Sort(core.BuildStatements(stmts))

		e.writeBuildFile(pkg, stmts)
	}
}

func (e *export) writeBuildFile(pkg *core.Package, stmts []core.BuildStatement) {
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

func (e *export) makeSubincludesStatement(pkg *core.Package) string {
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
