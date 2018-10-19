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

	<component name="libraryTable">
		<library name="third_party_java_gson">
			<CLASSES>
				<root contentUrl="jar://$PROJECT_DIR$/../../plz-out/gen/third_party/java/gson.jar!/"></root>
			</CLASSES>
			<JAVADOC></JAVADOC>
			<SOURCES>
				<root contentUrl="jar://$PROJECT_DIR$/../../plz-out/gen/third_party/java/gson_src.jar!/"></root>
			</SOURCES>
		</library>
	</component>

	<component name="libraryTable">
  		<library name="third_party_java_gson">
    		<CLASSES>
      			<root url="jar://$PROJECT_DIR$/../gen/third_party/java/gson.jar!/" />
    </CLASSES>
    <JAVADOC />
    <SOURCES>
      <root url="jar://$PROJECT_DIR$/../gen/third_party/java/gson_src.jar!/" />
    </SOURCES>
  </library>
</component>

	 */

 	os.RemoveAll(projectLocation())

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
	f.Close()

	// moduleTargets exist only for the modules we actually built, to keep the size down.
	moduleTargets := []*core.BuildTarget{}
	targetsToVisit := []*core.BuildTarget{}

	// For each target:
	for _, buildLabel := range originalLabels {
		targetsToVisit = append(targetsToVisit, graph.TargetOrDie(buildLabel))
	}

	visitTargetsAndDependenciesOnce(targetsToVisit, func(buildTarget *core.BuildTarget) {
		fmt.Println("Visiting", buildTarget.Label)
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
			m.toXml(f)
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

			library := NewLibrary(graph, buildTarget)
			library.toXml(f)
			f.Close()
		}
	})

	// Write modules.xml
	modules := NewModules(moduleTargets)
	if _, err := os.Stat(filepath.Dir(modulesFileLocation())); os.IsNotExist(err) {
		os.MkdirAll(filepath.Dir(modulesFileLocation()), core.DirPermissions)
	}
	f, err = os.Create(modulesFileLocation())
	if err != nil {
		log.Fatal("Unable to write modules file", err)
	}
	modules.toXml(f)
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