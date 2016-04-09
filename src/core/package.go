package core

import (
	"fmt"
	"sync"
)

// Representation of a package, ie. the part of the system (one or more
// directories) covered by a single build file.
type Package struct {
	// Name of the package, ie. //spam/eggs
	Name string
	// Filename of the build file that defined this package
	Filename string
	// Subincluded build defs files that this package imported
	Subincludes []string
	// Targets contained within the package
	Targets map[string]*BuildTarget
	// Set of output files from rules.
	Outputs map[string]*BuildTarget
	// Protects access to above
	Mutex sync.Mutex
}

func NewPackage(name string) *Package {
	pkg := new(Package)
	pkg.Name = name
	pkg.Targets = map[string]*BuildTarget{}
	pkg.Outputs = map[string]*BuildTarget{}
	return pkg
}

func (pkg *Package) RegisterSubinclude(filename string) {
	// Ensure these are unique.
	for _, fn := range pkg.Subincludes {
		if fn == filename {
			return
		}
	}
	pkg.Subincludes = append(pkg.Subincludes, filename)
}

// RegisterOutput registers a new output file in the map.
// Returns an error if the file has already been registered.
func (pkg *Package) RegisterOutput(fileName string, target *BuildTarget) error {
	pkg.Mutex.Lock()
	defer pkg.Mutex.Unlock()
	originalFileName := fileName
	if target.IsBinary {
		fileName = ":_bin_" + fileName // Add some arbitrary prefix so they don't clash.
	}
	if existing, present := pkg.Outputs[fileName]; present && existing != target {
		if target.HasSource(originalFileName) || existing.HasSource(originalFileName) {
			log.Debug("Rules %s and %s both output %s, but ignoring because we think one's a filegroup",
				existing.Label, target.Label, originalFileName)
		} else {
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
