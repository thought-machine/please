package core

import (
	"fmt"
	"path"
	"sort"
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
	Targets map[string]*BuildTarget
	// Set of output files from rules.
	Outputs map[string]*BuildTarget
	// Protects access to above
	mutex sync.Mutex
	// Used to arbitrate a single post-build function running at a time.
	// It would be sort of conceptually nice if they ran simultaneously but it'd
	// be far too hard to ensure consistency in any case where they can interact with one another.
	BuildCallbackMutex sync.Mutex
}

// NewPackage constructs a new package with the given name.
func NewPackage(name string) *Package {
	pkg := new(Package)
	pkg.Name = name
	pkg.Targets = map[string]*BuildTarget{}
	pkg.Outputs = map[string]*BuildTarget{}
	return pkg
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
		if existing.IsFilegroup() && !target.IsFilegroup() {
			// Update the existing one with this, the registered outputs should prefer non-filegroup targets.
			pkg.Outputs[fileName] = target
		} else if !target.IsFilegroup() && !existing.IsFilegroup() {
			return fmt.Errorf("Rules %s and %s in %s both attempt to output the same file: %s\n",
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
	for _, t := range pkg.Targets {
		if t.Label.Parent() == parent {
			ret = append(ret, t)
		}
	}
	sort.Sort(ret)
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
