package ide

import (
	"fmt"
	"path/filepath"
	"strings"

	"core"
)

// ExportIntellijStructure creates a set of modules and libraries that makes it nicer to work with Please projects
// in IntelliJ.
func ExportIntellijStructure(targets core.BuildTargets) {

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
		<content url="file://$MODULE_DIR$/../../../foo3">
		  <sourceFolder url="file://$MODULE_DIR$/../../../foo3" isTestSource="false" packagePrefix="foo3" />
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
	   <content url="file://$USER_HOME$/code/github.com/agenticarus/please" />
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
		 <sourceFolder url="file://$USER_HOME$/code/git.corp.tmachine.io/CORE/common/scala/phabricator" isTestSource="false" packagePrefix="common.scala.phabricator" />
	and library set to:
		<orderEntry type="library" name="finagle-base-http" level="project" />

	the library looks  like this:
	.idea/libraries/finagle_base_http.xml
   <component name="libraryTable">
	 <library name="finagle-base-http">
	   <CLASSES>
		 <root url="jar://$USER_HOME$/code/git.corp.tmachine.io/CORE/plz-out/gen/third_party/java/com/twitter/finagle-base-http.jar!/" />
	   </CLASSES>
	   <JAVADOC />
	   <SOURCES>
		 <root url="jar://$USER_HOME$/code/git.corp.tmachine.io/CORE/plz-out/gen/third_party/java/com/twitter/finagle-base-http_src.jar!/" />
	   </SOURCES>
	  </library>
   </component>%

	 */

	modules := make([]IModule, 0)

	for _, buildTarget := range targets {
		m := toModule(buildTarget)
		if m != nil {
			modules = append(modules, m)
		}
	}
}

func toModule(buildTarget *core.BuildTarget) IModule {
	for _, label := range buildTarget.PrefixedLabels("rule:") {
		if label == "java_library" {
			path := relativisedPathToPlzOutLocation(commonDirectoryFromInputs(buildTarget.Sources))
			if path != nil {
				return &JavaModule{
					Module: Module{
						url:          "file://$MODULE_DIR$/" + *path,
						isTestSource: false,
					},
					packagePrefix: packagePrefixFromLabels(buildTarget.PrefixedLabels("package_prefix:")),
				}
			}
		}
		if label == "java_test_library" {
			path := relativisedPathToPlzOutLocation(commonDirectoryFromInputs(buildTarget.Sources))
			return &JavaModule{
				Module: Module{
					url:          "file://$MODULE_DIR$/" + *path,
					isTestSource: true,
				},
				packagePrefix: nil, // or from label
			}
		}
		if label == "go_library" {
			path := relativisedPathToPlzOutLocation(commonDirectoryFromInputs(buildTarget.Sources))
			return &GoModule{
				Module: Module{
					url:          "file://$MODULE_DIR$/" + *path,
					isTestSource: false,
				},
			}
		}
	}
	return nil
}

func commonDirectoryFromInputs(inputs []core.BuildInput) *string {
	var commonPath *string = nil
	for _, input := range inputs {
		// Ignore systemfile things for now - they're the only ones that need a graph.
		for _, path := range input.Paths(nil) {
			fmt.Println(path)
			directory := filepath.Dir(path)
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

	return commonPath
}

func relativisedPathToPlzOutLocation(commonPath *string) *string {
	if commonPath == nil {
		return nil
	}

	rel, err := filepath.Rel("plz-out/intellij/"+*commonPath, *commonPath)
	if err == nil {
		return &rel
	}

	return nil
}

func packagePrefixFromLabels(labels []string) *string {
	if len(labels) != 1 {
		return nil
	}
	return &labels[0]
}

type Project struct {
	modules []IModule
}

type IModule interface {
	contentUrl() string
}

type Module struct {
	url          string
	isTestSource bool
}

func (module *Module) contentUrl() string {
	return module.url
}

type GoModule struct {
	Module
}

type JavaModule struct {
	Module
	packagePrefix *string
}
