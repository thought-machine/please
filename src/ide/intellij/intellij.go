package intellij

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/core"
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
	targetsToVisit := []core.BuildLabel{}

	// For each target:
	for _, buildLabel := range originalLabels {
		targetsToVisit = append(targetsToVisit, buildLabel)
	}

	visitTargetsAndDependenciesOnce(graph, targetsToVisit, func(label core.BuildLabel) {
		buildTarget := graph.TargetOrDie(label)
		m := toModule(graph, buildTarget)

		// Possibly write .iml
		if m != nil {
			if _, err := os.Stat(filepath.Dir(moduleFileLocation(label))); os.IsNotExist(err) {
				os.MkdirAll(filepath.Dir(moduleFileLocation(label)), core.DirPermissions)
			}
			f, err := os.Create(moduleFileLocation(label))
			if err != nil {
				log.Fatal("Unable to write module file for", label, "-", err)
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

func visitTargetsAndDependenciesOnce(graph *core.BuildGraph, original []core.BuildLabel, visitor func(label core.BuildLabel)) {
	targetsVisited := map[core.BuildLabel]core.BuildLabel{}
	for len(original) > 0 {
		var buildLabel core.BuildLabel
		buildLabel, original = original[0], original[1:]

		if _, ok := targetsVisited[buildLabel]; ok {
			continue
		}

		targetsVisited[buildLabel] = buildLabel

		visitor(buildLabel)

		original = append(original, graph.TargetOrDie(buildLabel).DeclaredDependencies()...)
	}
}

func outputLocation() string {
	return filepath.Join(core.RepoRoot, "plz-out", "intellij")
}

func projectLocation() string {
	return filepath.Join(outputLocation(), ".idea")
}

func moduleName(buildLabel core.BuildLabel) string {
	label := buildLabel.PackageName + "_" + buildLabel.Name
	label = strings.Replace(label, "/", "_", -1)
	label = strings.Replace(label, "#", "_", -1)
	return label
}

func moduleDirLocation(target *core.BuildTarget) string {
	return filepath.Join(outputLocation(), target.Label.PackageDir())
}

func moduleFileLocation(label core.BuildLabel) string {
	return filepath.Join(outputLocation(), label.PackageDir(), fmt.Sprintf("%s.iml", moduleName(label)))
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
