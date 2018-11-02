package intellij

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/op/go-logging.v1"

	"core"
)

var log = logging.MustGetLogger("intellij")

// ExportIntellijStructure creates a set of modules and libraries that makes it nicer to work with Please projects
// in IntelliJ.
func ExportIntellijStructure(config *core.Configuration, graph *core.BuildGraph, originalLabels core.BuildLabels) {
	os.RemoveAll(projectLocation())

	if _, err := os.Stat(projectLocation()); os.IsNotExist(err) {
		os.MkdirAll(projectLocation(), core.DirPermissions)
	}

	// Write misc.xml
	javaSourceLevel, err := strconv.Atoi(config.Java.SourceLevel)
	if err != nil {
		log.Fatal("Unable to determine java source level - ", err)
	}
	misc := newMisc(javaSourceLevel)

	f, err := os.Create(miscFileLocation())
	if err != nil {
		log.Fatal("Unable to create misc.xml - ", err)
	}
	misc.toXML(f)
	f.Close()

	// moduleTargets exist only for the modules we actually built, to keep the size down.
	moduleTargets := []*core.BuildTarget{}
	targetsToVisit := []*core.BuildTarget{}

	// For each target:
	for _, buildLabel := range originalLabels {
		targetsToVisit = append(targetsToVisit, graph.TargetOrDie(buildLabel))
	}

	visitTargetsAndDependenciesOnce(targetsToVisit, func(buildTarget *core.BuildTarget) {
		m := toModule(graph, buildTarget)

		// Possibly write .iml
		if m != nil {
			if _, err := os.Stat(filepath.Dir(moduleFileLocation(buildTarget))); os.IsNotExist(err) {
				os.MkdirAll(filepath.Dir(moduleFileLocation(buildTarget)), core.DirPermissions)
			}
			f, err := os.Create(moduleFileLocation(buildTarget))
			if err != nil {
				log.Fatal("Unable to write module file for", buildTarget.Label, "-", err)
			}
			m.toXML(f)
			f.Close()

			moduleTargets = append(moduleTargets, buildTarget)
		}

		// Possibly write library .xml
		if shouldMakeLibrary(buildTarget) {
			if _, err := os.Stat(libraryDirLocation()); os.IsNotExist(err) {
				os.MkdirAll(libraryDirLocation(), core.DirPermissions)
			}
			f, err := os.Create(libraryFileLocation(buildTarget))
			if err != nil {
				log.Fatal("Unable to write library file for", buildTarget.Label, "-", err)
			}

			library := newLibrary(graph, buildTarget)
			library.toXML(f)
			f.Close()
		}
	})

	// Write modules.xml
	modules := newModules(moduleTargets)
	if _, err := os.Stat(filepath.Dir(modulesFileLocation())); os.IsNotExist(err) {
		os.MkdirAll(filepath.Dir(modulesFileLocation()), core.DirPermissions)
	}
	f, err = os.Create(modulesFileLocation())
	if err != nil {
		log.Fatal("Unable to write modules file", err)
	}
	modules.toXML(f)
	f.Close()
}

func visitTargetsAndDependenciesOnce(original []*core.BuildTarget, visitor func(target *core.BuildTarget)) {
	targetsVisited := map[core.BuildLabel]core.BuildTarget{}
	for len(original) > 0 {
		var buildTarget *core.BuildTarget
		buildTarget, original = original[0], original[1:]

		if _, ok := targetsVisited[buildTarget.Label]; ok {
			continue
		}

		targetsVisited[buildTarget.Label] = *buildTarget

		visitor(buildTarget)

		original = append(original, buildTarget.Dependencies()...)
	}
}

func outputLocation() string {
	return filepath.Join(core.RepoRoot, "plz-out", "intellij")
}

func projectLocation() string {
	return filepath.Join(outputLocation(), ".idea")
}

func moduleName(target *core.BuildTarget) string {
	label := target.Label.PackageName + "_" + target.Label.Name
	label = strings.Replace(label, "/", "_", -1)
	label = strings.Replace(label, "#", "_", -1)
	return label
}

func moduleDirLocation(target *core.BuildTarget) string {
	return filepath.Join(outputLocation(), target.Label.PackageDir())
}

func moduleFileLocation(target *core.BuildTarget) string {
	return filepath.Join(outputLocation(), target.Label.PackageDir(), fmt.Sprintf("%s.iml", moduleName(target)))
}

func libraryDirLocation() string {
	return filepath.Join(projectLocation(), "libraries")
}

func libraryName(target *core.BuildTarget) string {
	label := target.Label.PackageName + "_" + target.Label.Name
	label = strings.Replace(label, "/", "_", -1)
	label = strings.Replace(label, ".", "_", -1)
	return label
}

func libraryFileLocation(target *core.BuildTarget) string {
	return filepath.Join(libraryDirLocation(), fmt.Sprintf("%s.xml", libraryName(target)))
}

func miscFileLocation() string {
	return filepath.Join(projectLocation(), "misc.xml")
}

func modulesFileLocation() string {
	return filepath.Join(projectLocation(), "modules.xml")
}
