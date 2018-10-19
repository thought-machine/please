package intellij

import (
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"

	"core"
)

/*
<?xml version="1.0" encoding="UTF-8"?>
<module type="JAVA_MODULE" version="4">
  <component name="NewModuleRootManager" inherit-compiler-output="true">
    <exclude-output />
    <orderEntry type="inheritedJdk" />
    <orderEntry type="sourceFolder" forTests="false" />
  </component>
</module>

<?xml version="1.0" encoding="UTF-8"?>
<module type="JAVA_MODULE" version="4">
  <component name="NewModuleRootManager" inherit-compiler-output="true">
    <exclude-output />
    <content url="file://$MODULE_DIR$">
      <sourceFolder url="file://$MODULE_DIR$/src" isTestSource="false" />
    </content>
    <orderEntry type="inheritedJdk" />
    <orderEntry type="sourceFolder" forTests="false" />
    <orderEntry type="module" module-name="foo2" />
  </component>
</module>

	<orderEntry type="library" name="finagle-base-http" level="project" />

*/

type Module struct {
	XMLName    xml.Name          `xml:"module"`
	ModuleType string            `xml:"type,attr"`
	Version    int               `xml:"version,attr"`
	Component  []ModuleComponent `xml:"component"`
}

func NewJavaModule(graph *core.BuildGraph, target *core.BuildTarget) Module {
	component := NewModuleComponent(graph, target)
	component.addOrderEntry(NewInheritedJdkEntry())
	component.addOrderEntry(NewSourceFolderEntry(false))

	for _, label := range target.DeclaredDependencies() {
		dep := graph.TargetOrDie(label)
		if shouldMakeModule(dep) {
			component.addOrderEntry(NewModuleEntry(moduleName(dep)))
		}
	}

	if shouldMakeLibrary(target) {
		component.addOrderEntry(NewLibraryEntry(libraryName(target), "project"))
	}

	module := Module{
		ModuleType: "JAVA_MODULE",
		Version:    4,
		Component: []ModuleComponent{
			component,
		},
	}
	return module
}

func NewScalaModule(graph *core.BuildGraph, target *core.BuildTarget) Module {
	module := NewJavaModule(graph, target)

	module.Component[0].addOrderEntry(NewLibraryEntry("scala-sdk", "application"))

	return module
}

func shouldMakeJavaModule(target *core.BuildTarget) bool {
	for _, label := range target.PrefixedLabels("rule:") {
		if label == "java_library" {
			return true
		} else if label == "java_test" {
			return true
		} else if label == "maven_jar" {
			return true
		}
	}
	return false
}

func shouldMakeScalaModule(target *core.BuildTarget) bool {
	for _, label := range target.PrefixedLabels("rule:") {
		if label == "scala_library" {
			return true
		}
	}
	return false
}

func shouldMakeModule(target *core.BuildTarget) bool {
	return shouldMakeJavaModule(target) ||
		shouldMakeScalaModule(target)
}

func shouldHaveContent(target *core.BuildTarget) bool {
	for _, label := range target.PrefixedLabels("rule:") {
		if label == "java_library" {
			return true
		} else if label == "java_test" {
			return true
		} else if label == "scala_library" {
			return true
		}
	}
	return false
}

func shouldMakeLibrary(target *core.BuildTarget) bool {
	for _, label := range target.PrefixedLabels("rule:") {
		if label == "maven_jar" {
			return true
		}
	}
	return false
}

func isTestModule(target *core.BuildTarget) bool {
	if target.IsTest {
		return true
	}
	for _, label := range target.PrefixedLabels("rule:") {
		if label == "java_test" {
			return true
		}
	}
	return false
}

func NewWebModule(graph *core.BuildGraph, target *core.BuildTarget) Module {
	return Module{
		ModuleType: "WEB_MODULE",
		Version:    4,
		Component: []ModuleComponent{
			NewModuleComponent(graph, target),
		},
	}
}

func (module *Module) toXml(writer io.Writer) {
	writer.Write([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"))

	content, err := xml.MarshalIndent(module, "", "  ")
	if err  == nil {
		writer.Write(content)
	}
}

type ModuleComponent struct {
	XMLName               xml.Name       `xml:"component"`
	Name                  string         `xml:"name,attr"`
	InheritCompilerOutput bool           `xml:"inherit-compiler-output,attr"`
	Content               *ModuleContent `xml:"content,omitEmpty"`
	OrderEntries          []OrderEntry   `xml:"orderEntry"`
}

func NewModuleComponent(graph *core.BuildGraph, target *core.BuildTarget) ModuleComponent {
	orderEntries := []OrderEntry{}

	return ModuleComponent{
		Name: "NewModuleRootManager",
		InheritCompilerOutput: true,
		Content:               NewModuleContent(graph, target),
		OrderEntries:          orderEntries,
	}
}

func (moduleComponent *ModuleComponent) addOrderEntry(entry OrderEntry) {
	moduleComponent.OrderEntries = append(moduleComponent.OrderEntries, entry)
}

type ModuleContent struct {
	XMLName      xml.Name       `xml:"content"`
	Url          string         `xml:"url,attr"`
	SourceFolder []SourceFolder `xml:"sourceFolder"`
}

func NewModuleContent(graph *core.BuildGraph, target *core.BuildTarget) *ModuleContent {
	if !shouldHaveContent(target) {
		return nil
	}

	sourceFolders := []SourceFolder{}
	commonDir := target.Label.PackageDir()

	joined := filepath.Join(core.RepoRoot, commonDir)
	location := relativisedPathTo(moduleDirLocation(target), &joined)

	sourceFolders = append(sourceFolders, SourceFolder{
		Url:           fmt.Sprintf("file://$MODULE_DIR$/%s", *location),
		IsTestSource:  isTestModule(target),
		PackagePrefix: packagePrefixFromLabels(target.PrefixedLabels("package_prefix:")),
	})

	packageDir := commonDir
	maybeSrcDir := target.PrefixedLabels("src_dir:")
	if len(maybeSrcDir) == 1 {
		packageDir = filepath.Join(packageDir, maybeSrcDir[0])
	}
	joined = filepath.Join(core.RepoRoot, packageDir)
	contentDir := relativisedPathTo(moduleDirLocation(target), &joined)

	return &ModuleContent{
		Url:          fmt.Sprintf("file://$MODULE_DIR$/%s", *contentDir),
		SourceFolder: sourceFolders,
	}
}

type SourceFolder struct {
	XMLName       xml.Name `xml:"sourceFolder"`
	Url           string   `xml:"url,attr"`
	IsTestSource  bool     `xml:"isTestSource,attr"`
	PackagePrefix *string  `xml:"packagePrefix,attr,omitEmpty"`
}

type OrderEntry struct {
	XMLName xml.Name `xml:"orderEntry"`
	Type    string   `xml:"type,attr"`

	ForTests *bool `xml:"forTests,attr,omitEmpty"`

	Exported *string `xml:"exported,attr,omitEmpty"`

	LibraryName  *string `xml:"name,attr,omitEmpty"`
	LibraryLevel *string `xml:"level,attr,omitEmpty"`

	ModuleName *string `xml:"module-name,attr,omitEmpty"`
}

func NewLibraryEntry(name, level string) OrderEntry {
	exported := ""
	return OrderEntry{
		Type:         "library",
		LibraryName:  &name,
		LibraryLevel: &level,
		Exported: &exported,
	}
}

func NewModuleEntry(name string) OrderEntry {
	exported := ""
	return OrderEntry{
		Type:       "module",
		ModuleName: &name,
		Exported: &exported,
	}
}

func NewInheritedJdkEntry() OrderEntry {
	return OrderEntry{
		Type: "inheritedJdk",
	}
}

func NewSourceFolderEntry(forTests bool) OrderEntry {
	return OrderEntry{
		Type:     "sourceFolder",
		ForTests: &forTests,
	}
}

func toModule(graph *core.BuildGraph, buildTarget *core.BuildTarget) *Module {
	var module *Module = nil

	if shouldMakeJavaModule(buildTarget) {
		madeModule := NewJavaModule(graph, buildTarget)
		module = &madeModule
	}

	if shouldMakeScalaModule(buildTarget) {
		madeModule := NewScalaModule(graph, buildTarget)
		module = &madeModule
	}

	return module
}

func relativisedPathTo(location string, commonPath *string) *string {
	if commonPath == nil {
		return nil
	}

	rel, err := filepath.Rel(location, *commonPath)
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
