package core

import (
	"fmt"
	"path"
	"sort"
	"strings"
	"sync"
)

// Package is a representation of a package, ie. the part of the system (one or more
// directories) covered by a single build file.
type Package struct {
	// Name of the package, ie. //spam/eggs
	Name string
	// Filename of the build file that defined this package
	Filename string
	// Subincluded build defs files that this package imported
	Subincludes []BuildLabel
	// Targets contained within the package
	targets map[string]*BuildTarget
	// Set of output files from rules.
	Outputs map[string]*BuildTarget
	// Protects access to above
	mutex sync.RWMutex
	// Targets whose dependencies got modified during a pre or post-build function.
	modifiedTargets map[*BuildTarget]struct{}
	// Used to arbitrate a single post-build function running at a time.
	// It would be sort of conceptually nice if they ran simultaneously but it'd
	// be far too hard to ensure consistency in any case where they can interact with one another.
	buildCallbackMutex sync.Mutex
}

// NewPackage constructs a new package with the given name.
func NewPackage(name string) *Package {
	return &Package{
		Name:    name,
		targets: map[string]*BuildTarget{},
		Outputs: map[string]*BuildTarget{},
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

// HasOutput returns true if the package has the given file as an output.
func (pkg *Package) HasOutput(output string) bool {
	pkg.mutex.RLock()
	defer pkg.mutex.RUnlock()
	_, present := pkg.Outputs[output]
	return present
}

// RegisterOutput registers a new output file in the map.
// Returns an error if the file has already been registered.
func (pkg *Package) RegisterOutput(fileName string, target *BuildTarget) error {
	pkg.mutex.Lock()
	defer pkg.mutex.Unlock()
	originalFileName := fileName
	if target.IsBinary {
		fileName = ":_bin_" + fileName // Add some arbitrary prefix so they don't clash.
	}
	if existing, present := pkg.Outputs[fileName]; present && existing != target {
		if existing.IsFilegroup && !target.IsFilegroup {
			// Update the existing one with this, the registered outputs should prefer non-filegroup targets.
			pkg.Outputs[fileName] = target
		} else if !target.IsFilegroup && !existing.IsFilegroup {
			return fmt.Errorf("rules %s and %s in %s both attempt to output the same file: %s",
				existing.Label, target.Label, pkg.Filename, originalFileName)
		}
	}
	pkg.Outputs[fileName] = target
	return nil
}

// MustRegisterOutput registers a new output file and panics if it's already been registered.
func (pkg *Package) MustRegisterOutput(fileName string, target *BuildTarget) {
	if err := pkg.RegisterOutput(fileName, target); err != nil {
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

// EnterBuildCallback is used to arbitrate access to build callbacks & track changes to targets.
// The supplied function will be called & a set of modified targets, along with any errors, is returned.
func (pkg *Package) EnterBuildCallback(f func() error) (map[*BuildTarget]struct{}, error) {
	pkg.buildCallbackMutex.Lock()
	defer pkg.buildCallbackMutex.Unlock()
	m := map[*BuildTarget]struct{}{}
	pkg.modifiedTargets = m
	err := f()
	pkg.modifiedTargets = nil
	return m, err
}

// MarkTargetModified marks a single target as being modified during a pre- or post- build function.
// Correct usage of EnterBuildCallback must have been observed.
func (pkg *Package) MarkTargetModified(target *BuildTarget) {
	if pkg.modifiedTargets != nil {
		pkg.modifiedTargets[target] = struct{}{}
	}
}

// VerifyOutputs checks all files output from this package and verifies that they're all OK;
// notably it checks that if targets that output into a subdirectory, that subdirectory isn't
// created by another target. That kind of thing can lead to subtle and annoying bugs.
// It logs detected warnings to stdout.
func (pkg *Package) VerifyOutputs() {
	for _, warning := range pkg.verifyOutputs() {
		log.Warning("%s: %s", pkg.Filename, warning)
	}
}

func (pkg *Package) verifyOutputs() []string {
	pkg.mutex.RLock()
	defer pkg.mutex.RUnlock()
	ret := []string{}
	for filename, target := range pkg.Outputs {
		for dir := path.Dir(filename); dir != "."; dir = path.Dir(dir) {
			if target2, present := pkg.Outputs[dir]; present && target2 != target && !target.HasDependency(target2.Label.Parent()) {
				ret = append(ret, fmt.Sprintf("Target %s outputs files into the directory %s, which is separately output by %s. This can cause errors based on build order - you should add a dependency.", target.Label, dir, target2.Label))
			}
		}
	}
	return ret
}

// FindOwningPackages returns build labels corresponding to the packages that own each of the given files.
func FindOwningPackages(files []string) []BuildLabel {
	ret := make([]BuildLabel, len(files))
	for i, file := range files {
		ret[i] = FindOwningPackage(file)
	}
	return ret
}

// FindOwningPackage returns a build label identifying the package that owns a given file.
func FindOwningPackage(file string) BuildLabel {
	f := file
	for f != "." {
		f = path.Dir(f)
		if IsPackage(f) {
			return BuildLabel{PackageName: f, Name: "all"}
		}
	}
	log.Fatalf("No BUILD file owns file %s", file)
	return BuildLabel{}
}
