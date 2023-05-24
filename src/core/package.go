package core

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/fs"
)

// Max levenshtein distance that we'll suggest at.
const maxSuggestionDistance = 3

// Package is a representation of a package, ie. the part of the system (one or more
// directories) covered by a single build file.
type Package struct {
	// Name of the package, ie. //spam/eggs
	Name string
	// If this package is in a subrepo, this is the name of the subrepo.
	// Equivalent to Subrepo.Name but avoids NPEs.
	SubrepoName string
	// Filename of the build file that defined this package
	Filename string
	// Subincluded build defs files that this package imported
	Subincludes []BuildLabel
	// If the package is in a subrepo, this is the subrepo it belongs to. It's nil if not.
	Subrepo *Subrepo
	// Targets contained within the package
	targets map[string]*BuildTarget
	// Set of output files from rules.
	Outputs map[string]*BuildTarget
	// Protects access to above
	mutex sync.RWMutex
}

// NewPackage constructs a new package with the given name.
func NewPackage(name string) *Package {
	return NewPackageSubrepo(name, "")
}

// NewPackageSubrepo constructs a new package with the given name and subrepo.
func NewPackageSubrepo(name, subrepo string) *Package {
	return &Package{
		Name:        name,
		SubrepoName: subrepo,
		targets:     map[string]*BuildTarget{},
		Outputs:     map[string]*BuildTarget{},
	}
}

// Target returns the target with the given name, or nil if this package doesn't have one.
func (pkg *Package) Target(name string) *BuildTarget {
	pkg.mutex.RLock()
	defer pkg.mutex.RUnlock()
	return pkg.targets[name]
}

// TargetOrDie returns the target with the given name, and dies if this package doesn't have one.
func (pkg *Package) TargetOrDie(name string) *BuildTarget {
	t := pkg.Target(name)
	if t == nil {
		log.Fatalf("Target %s not registered in package %s", name, pkg.Name)
	}
	return t
}

// SourceRoot returns the root directory of source files for this package.
// This is equivalent to .Name for in-repo packages but differs for those in subrepos.
func (pkg *Package) SourceRoot() string {
	if pkg.Subrepo != nil {
		return pkg.Subrepo.Dir(pkg.Name)
	}
	return pkg.Name
}

// AddTarget adds a new target to this package with the given name.
// It doesn't check for duplicates.
func (pkg *Package) AddTarget(target *BuildTarget) {
	pkg.mutex.Lock()
	defer pkg.mutex.Unlock()
	pkg.targets[target.Label.Name] = target
}

// AllTargets returns the current set of targets in this package.
// This is threadsafe, unlike iterating .Targets directly which is not.
func (pkg *Package) AllTargets() []*BuildTarget {
	pkg.mutex.Lock()
	defer pkg.mutex.Unlock()
	ret := make([]*BuildTarget, 0, len(pkg.targets))
	for _, target := range pkg.targets {
		ret = append(ret, target)
	}
	return ret
}

// NumTargets returns the number of targets currently registered in this package.
func (pkg *Package) NumTargets() int {
	pkg.mutex.Lock()
	defer pkg.mutex.Unlock()
	return len(pkg.targets)
}

// RegisterSubinclude adds a new subinclude to this package, guaranteeing uniqueness.
func (pkg *Package) RegisterSubinclude(label BuildLabel) {
	if !pkg.HasSubinclude(label) {
		pkg.Subincludes = append(pkg.Subincludes, label)
	}
}

// HasSubinclude returns true if the package has subincluded the given label.
func (pkg *Package) HasSubinclude(label BuildLabel) bool {
	for _, l := range pkg.Subincludes {
		if l == label {
			return true
		}
	}
	return false
}

// SubrepoArchName returns a subrepo name, modified for the architecture of this package if it's not the host.
func (pkg *Package) SubrepoArchName(subrepo string) string {
	if subrepo != "" && pkg.Subrepo != nil && pkg.Subrepo.IsCrossCompile && pkg.SubrepoName != subrepo {
		return SubrepoArchName(subrepo, pkg.Subrepo.Arch)
	}
	return subrepo
}

// HasOutput returns true if the package has the given file as an output.
func (pkg *Package) HasOutput(output string) bool {
	pkg.mutex.RLock()
	defer pkg.mutex.RUnlock()
	_, present := pkg.Outputs[output]
	return present
}

// RegisterOutput registers a new output file in the map.
// Returns an error if the file has already been registered.
func (pkg *Package) RegisterOutput(state *BuildState, fileName string, target *BuildTarget) error {
	pkg.mutex.Lock()
	defer pkg.mutex.Unlock()

	originalFileName := fileName
	if target.IsBinary {
		fileName = ":_bin_" + fileName // Add some arbitrary prefix so they don't clash.
	}

	if existing, present := pkg.Outputs[fileName]; present && existing != target {
		// Only local files are available as outputs to filegroups at this stage, so unless both targets are filegroups
		// then the same output isn't allowed.
		if !target.IsFilegroup || !existing.IsFilegroup {
			return fmt.Errorf("rules %s and %s in %s both attempt to output the same file: %s", existing.Label, target.Label, pkg.Filename, originalFileName)
		}
	}

	pkg.Outputs[fileName] = target

	return nil
}

// MustRegisterOutput registers a new output file and panics if it's already been registered.
func (pkg *Package) MustRegisterOutput(state *BuildState, fileName string, target *BuildTarget) {
	if err := pkg.RegisterOutput(state, fileName, target); err != nil {
		panic(err)
	}
}

// AllChildren returns all child targets of the given one.
// The given target is included as well.
func (pkg *Package) AllChildren(target *BuildTarget) []*BuildTarget {
	ret := BuildTargets{}
	parent := target.Label.Parent()
	for _, t := range pkg.targets {
		if t.Label.Parent() == parent {
			ret = append(ret, t)
		}
	}
	sort.Sort(ret)
	return ret
}

// IsIncludedIn returns true if the given build label would include this package.
// e.g. //src/... includes the packages src and src/core but not src2.
func (pkg *Package) IsIncludedIn(label BuildLabel) bool {
	return pkg.Name == label.PackageName || strings.HasPrefix(pkg.Name, label.PackageName+"/")
}

// Label returns a build label uniquely identifying this package.
func (pkg *Package) Label() BuildLabel {
	return BuildLabel{Subrepo: pkg.SubrepoName, PackageName: pkg.Name, Name: "all"}
}

// MustVerifyOutputs checks all files output from this package and verifies that they're all OK;
// notably it checks that if targets that output into a subdirectory, that subdirectory isn't
// created by another target. That kind of thing can lead to subtle and annoying bugs.
func (pkg *Package) MustVerifyOutputs() {
	if issues := pkg.verifyOutputs(); len(issues) > 0 {
		log.Fatalf("%s: %s", pkg.Filename, issues[0])
	}
}

// It logs detected issues as warnings to stdout.
func (pkg *Package) VerifyOutputs() {
	for _, issue := range pkg.verifyOutputs() {
		log.Warning("%s: %s", pkg.Filename, issue)
	}
}

func (pkg *Package) verifyOutputs() []string {
	pkg.mutex.RLock()
	defer pkg.mutex.RUnlock()
	ret := []string{}
	for filename, target := range pkg.Outputs {
		for dir := filepath.Dir(filename); dir != "."; dir = filepath.Dir(dir) {
			if target2, present := pkg.Outputs[dir]; present && target2 != target && !(target.HasDependency(target2.Label.Parent()) || target.HasDependency(target2.Label)) {
				ret = append(ret, fmt.Sprintf("Target %s outputs files into the directory %s, which is separately output by %s. This can cause errors based on build order - you should add a dependency.", target.Label, dir, target2.Label))
			}
		}
	}
	return ret
}

// FindOwningPackages returns build labels corresponding to the packages that own each of the given files.
func FindOwningPackages(state *BuildState, files []string) []BuildLabel {
	ret := make([]BuildLabel, len(files))
	for i, file := range files {
		ret[i] = FindOwningPackage(state, file)
		if ret[i].PackageName == "" {
			log.Fatalf("No BUILD file owns file %s", file)
		}
	}
	return ret
}

// FindOwningPackage returns a build label identifying the package that owns a given file.
func FindOwningPackage(state *BuildState, file string) BuildLabel {
	f := filepath.Dir(file)
	for f != "." {
		if fs.IsPackage(state.Config.Parse.BuildFileName, f) {
			return BuildLabel{PackageName: f, Name: "all"}
		}
		f = filepath.Dir(f)
	}
	return BuildLabel{PackageName: "", Name: "all"}
}

// suggestTargets suggests the targets in the given package that might be misspellings of
// the requested one.
func suggestTargets(pkg *Package, label, dependent BuildLabel) string {
	// The initial haystack only contains target names
	haystack := []string{}
	for _, t := range pkg.AllTargets() {
		haystack = append(haystack, fmt.Sprintf("//%s:%s", pkg.Name, t.Label.Name))
	}
	msg := cli.PrettyPrintSuggestion(label.String(), haystack, maxSuggestionDistance)
	if pkg.Name != dependent.PackageName {
		return msg
	}
	// Use relative package labels where possible.
	return strings.ReplaceAll(msg, "//"+pkg.Name+":", ":")
}
