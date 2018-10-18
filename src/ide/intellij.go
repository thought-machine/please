package ide

import (
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"core"
)

// ExportIntellijStructure creates a set of modules and libraries that makes it nicer to work with Please projects
// in IntelliJ.
func ExportIntellijStructure(graph *core.BuildGraph, targets core.BuildTargets) {

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

	modules := make([]IModule, 0)

	for _, buildTarget := range targets {
		m := toModule(graph, buildTarget)
		//if m != nil {
			for _, library := range findLibraries(graph, buildTarget) {
				m.addLibrary(library)
			}
			modules = append(modules, m)
		//}
	}
}

func toModule(graph *core.BuildGraph, buildTarget *core.BuildTarget) IModule {
	for _, label := range buildTarget.PrefixedLabels("rule:") {
		if label == "java_library" {
			path := relativisedPathToPlzOutLocation(commonDirectoryFromInputs(graph, buildTarget.Sources))
			if path != nil {
				return &JavaModule{
					Module: Module{
						contentUrl:   "file://$MODULE_DIR$/" + *path,
						isTestSource: false,
					},
					packagePrefix: packagePrefixFromLabels(buildTarget.PrefixedLabels("package_prefix:")),
				}
			}
		}
		if label == "java_test_library" {
			path := relativisedPathToPlzOutLocation(commonDirectoryFromInputs(graph, buildTarget.Sources))
			return &JavaModule{
				Module: Module{
					contentUrl:   "file://$MODULE_DIR$/" + *path,
					isTestSource: true,
				},
				packagePrefix: nil, // or from label
			}
		}
	}
	return nil
}

func findLibraries(graph *core.BuildGraph, target *core.BuildTarget) []Library {
	libraries := make([]Library, 0)

	fmt.Printf("Found %d dependencies for %s\n", len(target.Dependencies()), target.Label)
	return libraries
}

func commonDirectoryFromInputs(graph *core.BuildGraph, inputs []core.BuildInput) *string {
	var commonPath *string = nil
	for _, input := range inputs {
		for _, path := range input.Paths(graph) {
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
	if err != nil {
		return nil
	}

	return &rel
}

func packagePrefixFromLabels(labels []string) *string {
	if len(labels) != 1 {
		return nil
	}
	return &labels[0]
}

type IProject interface{}

type project struct {
	version int
	modules []IModule
}

func NewProject() IProject {
	return &project{
		version: 4,
		modules: make([]IModule, 0),
	}
}

type IModule interface {
	addLibrary(library Library)
}

type Module struct {
	contentUrl   string
	isTestSource bool
	libraries    []Library
}

func (module *Module) addLibrary(library Library) {
	module.libraries = append(module.libraries, library)
}

type JavaModule struct {
	Module
	packagePrefix *string
}

type libraryComponent struct {
	XMLName xml.Name `xml:"component"`
	Name string `xml:"name,attr"`
	Library Library `xml:"library"`
}

type Content struct {
	XMLName    xml.Name `xml:"root"`
	ContentUrl string   `xml:"contentUrl,attr"`
}

type Library struct {
	XMLName      xml.Name  `xml:"library"`
	Name         string    `xml:"name,attr"`
	ClassPaths   []Content `xml:"CLASSES>root"`
	JavadocPaths []Content `xml:"JAVADOC>root"`
	SourcePaths  []Content `xml:"SOURCES>root"`
}

func (library *Library) toXml(writer io.Writer) {
	encoder := xml.NewEncoder(writer)
	encoder.EncodeToken(xml.ProcInst{Target:"xml", Inst: []byte("version=\"1.0\" encoding=\"UTF-8\"")})

	table := &libraryComponent{
		Name: "libraryTable",
		Library: *library,
	}
	encoder.Encode(table)
}
