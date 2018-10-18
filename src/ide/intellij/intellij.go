package intellij

import (
	"fmt"
	"gopkg.in/op/go-logging.v1"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"core"
)

var log = logging.MustGetLogger("intellij")

// ExportIntellijStructure creates a set of modules and libraries that makes it nicer to work with Please projects
// in IntelliJ.
func ExportIntellijStructure(config *core.Configuration, graph *core.BuildGraph, targets core.BuildTargets, originalLabels core.BuildLabels) {

	// Structure is as follows:
	/*
	Top level: .idea/modules.xml
	<?xml version="1.0" encoding="UTF-8"?>
	<project version="4">
	  <component name="ProjectModuleManager">
		<modules>
		  <module fileurl="file://$PROJECT_DIR$/foo/foo.iml" filepath="$PROJECT_DIR$/foo/foo.iml" />
		  <module fileurl="file://$PROJECT_DIR$/foo2/foo2.iml" filepath="$PROJECT_DIR$/foo2/foo2.iml" />
		  <module fileurl="file://$PROJECT_DIR$/plz-out/gen/intellij/foo3.iml" filepath="$PROJECT_DIR$/plz-out/gen/intellij/foo3.iml" />
		</modules>
	  </component>
	</project>
	*/
	/*
	Java:
	Module file: ./plz-out/gen/intellij/foo3.iml
	<?xml version="1.0" encoding="UTF-8"?>
	<module type="JAVA_MODULE" version="4">
	  <component name="NewModuleRootManager" inherit-compiler-output="true">
		<exclude-output />
		<content contentUrl="file://$MODULE_DIR$/../../../foo3">
		  <sourceFolder contentUrl="file://$MODULE_DIR$/../../../foo3" isTestSource="false" packagePrefix="foo3" />
		</content>
		<orderEntry type="inheritedJdk" />
		<orderEntry type="sourceFolder" forTests="false" />
	  </component>
	</module>
 	*/
	/*
	Go:
	Module file:
	<?xml version="1.0" encoding="UTF-8"?>
   <module type="WEB_MODULE" version="4">
	 <component name="Go" enabled="true" />
	 <component name="NewModuleRootManager">
	   <content contentUrl="file://$USER_HOME$/code/github.com/agenticarus/please" />
	   <orderEntry type="inheritedJdk" />
	   <orderEntry type="sourceFolder" forTests="false" />
	 </component>
   </module>

	Workspace: workspace.xml
   <?xml version="1.0" encoding="UTF-8"?>
   <project version="4">
	...
   <component name="GOROOT" path="/opt/tm/tools/go/1.10.2/usr/go" />
	<component name="GoLibraries">
		<option name="urls">
		 <list>
		   <option value="file://$USER_HOME$/code/github.com/agenticarus/please/src" />
		   <option value="file://$USER_HOME$/code/github.com/agenticarus/please/plz-out/go" />
		 </list>
	   </option>
	 </component>
	...
	</project>
	*/

	/*
	Scala:
	need <orderEntry type="library" name="scala-sdk-2.12.7" level="application" /> in modules.xml

	//common/scala/phabricator:BUILD contains one scala_library
	should map to one module with source folder set to:
		 <sourceFolder contentUrl="file://$USER_HOME$/code/git.corp.tmachine.io/CORE/common/scala/phabricator" isTestSource="false" packagePrefix="common.scala.phabricator" />
	and library set to:
		<orderEntry type="library" name="finagle-base-http" level="project" />

	the library looks  like this:
	.idea/libraries/finagle_base_http.xml
   <component name="libraryTable">
	 <library name="finagle-base-http">
	   <CLASSES>
		 <root contentUrl="jar://$USER_HOME$/code/git.corp.tmachine.io/CORE/plz-out/gen/third_party/java/com/twitter/finagle-base-http.jar!/" />
	   </CLASSES>
	   <JAVADOC />
	   <SOURCES>
		 <root contentUrl="jar://$USER_HOME$/code/git.corp.tmachine.io/CORE/plz-out/gen/third_party/java/com/twitter/finagle-base-http_src.jar!/" />
	   </SOURCES>
	  </library>
   </component>%

	 */

	if _, err := os.Stat(projectLocation()); os.IsNotExist(err) {
		os.MkdirAll(projectLocation(), core.DirPermissions)
	}

	// Write misc.xml

	javaSourceLevel, err := strconv.Atoi(config.Java.SourceLevel)
	if err != nil {
		log.Fatal("Unable to determine java source level - ", err)
	}
	misc := NewMisc(javaSourceLevel)

	f, err := os.Create(miscFileLocation())
	if err != nil {
		log.Fatal("Unable to create misc.xml - ", err)
	}
	misc.toXml(f)

	// For each target:
	for _, buildTarget := range targets {
		m, l := toModuleAndLibrary(graph, buildTarget)

		if _, err := os.Stat(filepath.Dir(moduleFileLocation(buildTarget))); os.IsNotExist(err) {
			os.MkdirAll(filepath.Dir(moduleFileLocation(buildTarget)), core.DirPermissions)
		}
		f, err := os.Create(moduleFileLocation(buildTarget))
		if err != nil {
			log.Fatal("Unable to write module file for ", buildTarget.Label, " - ", err)
		}
		// Write .iml
		m.toXml(f)
		// Possibly write libraries .xml
		if l != nil {
			if _, err := os.Stat(libraryDirLocation()); os.IsNotExist(err) {
				os.MkdirAll(libraryDirLocation(), core.DirPermissions)
			}
			f, err := os.Create(moduleFileLocation(buildTarget))
			if err != nil {
				log.Fatal("Unable to write library file for", buildTarget.Label, "-", err)
			}
			l.toXml(f)
		}
	}

	// Write modules.xml
	//originalTargets := []*core.BuildTarget{}
	//for _, label := range originalLabels {
	//	originalTargets = append(originalTargets, graph.TargetOrDie(label))
	//}
	modules := NewModules(targets)
	if _, err := os.Stat(filepath.Dir(modulesFileLocation())); os.IsNotExist(err) {
		os.MkdirAll(filepath.Dir(modulesFileLocation()), core.DirPermissions)
	}
	f, err = os.Create(modulesFileLocation())
	if err != nil {
		log.Fatal("Unable to write modules file", err)
	}
	// Write .iml
	modules.toXml(f)
}

func commonDirectoryFromInputs(graph *core.BuildGraph, inputs []core.BuildInput) *string {
	var commonPath *string = nil
	for _, input := range inputs {
		for _, path := range input.Paths(graph) {
			if f, err := os.Stat(path); err == nil {
				directory := filepath.Dir(path)
				if f.Mode().IsDir() {
					directory = path
				}
				if commonPath == nil {
					commonPath = &directory
				} else {
					for {
						// resolve against the current common path
						rel, err := filepath.Rel(*commonPath, directory);
						if err != nil {
							return nil
						}

						if strings.HasPrefix(rel, "..") {
							parent := filepath.Dir(*commonPath)
							commonPath = &parent
							continue
						}

						break
					}
				}
			}
		}
	}

	return commonPath
}

func outputLocation() string {
	return filepath.Join(core.RepoRoot, "plz-out", "intellij")
}

func projectLocation() string {
	return filepath.Join(outputLocation(), ".idea")
}

func moduleName(target *core.BuildTarget) string {
	return strings.Replace(target.Label.PackageName + "_" + target.Label.Name, "/", "_", -1)
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
	// This is currently the same as moduleName but there is no requirement for that, and indeed we may want to use the version here too.
	return strings.Replace(target.Label.PackageName + "_" + target.Label.Name, "/", "_", -1)
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