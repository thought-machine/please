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
	XMLName      xml.Name `xml:"module"`
	ModuleType   string   `xml:"type,attr"`
	Version      int      `xml:"version,attr"`
	Component    []ModuleComponent `xml:"component"`
}

func NewJavaModule(graph *core.BuildGraph, target *core.BuildTarget) Module {
	component := NewModuleComponent(graph, target)
	component.addOrderEntry(NewInheritedJdkEntry())
	component.addOrderEntry(NewSourceFolderEntry(false))

	for _, dep := range target.Dependencies() {
		component.addOrderEntry(NewModuleEntry(moduleName(dep)))
	}

	return Module{
		ModuleType:   "JAVA_MODULE",
		Version:      4,
		Component:    []ModuleComponent{
			component,
		},
	}
}

func NewWebModule(graph *core.BuildGraph, target *core.BuildTarget) Module {
	return Module{
		ModuleType:   "WEB_MODULE",
		Version:      4,
		Component:    []ModuleComponent{
			NewModuleComponent(graph, target),
		},
	}
}

func (module *Module) toXml(writer io.Writer) {
	encoder := xml.NewEncoder(writer)
	encoder.EncodeToken(xml.ProcInst{Target:"xml", Inst: []byte("version=\"1.0\" encoding=\"UTF-8\"")})

	encoder.Encode(module)
}

type ModuleComponent struct {
	XMLName               xml.Name      `xml:"component"`
	Name                  string        `xml:"name,attr"`
	InheritCompilerOutput bool          `xml:"inherit-compiler-output,attr"`
	Content               *ModuleContent `xml:"content,omitEmpty"`
	OrderEntries          []OrderEntry  `xml:"orderEntry"`
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
	commonDir := commonDirectoryFromInputs(graph, target.Sources)

	if commonDir == nil {
		return nil
	}

	joined := filepath.Join(core.RepoRoot, *commonDir)
	path := relativisedPathTo(moduleDirLocation(target), &joined)

	sourceFolders := []SourceFolder{
		{
			Url: fmt.Sprintf("file://$MODULE_DIR$/%s", *path),
			ForTests: false,
			PackagePrefix: packagePrefixFromLabels(target.PrefixedLabels("package_prefix:")),
		},
	}

	packageDir := target.Label.PackageDir()
	joined = filepath.Join(core.RepoRoot, packageDir)
	contentDir := relativisedPathTo(moduleDirLocation(target), &joined)

	return &ModuleContent{
		Url: fmt.Sprintf("file://$MODULE_DIR$/%s", *contentDir),
		SourceFolder: sourceFolders,
	}
}

type SourceFolder struct {
	XMLName       xml.Name `xml:"sourceFolder"`
	Url           string   `xml:"url,attr"`
	ForTests      bool     `xml:"forTests,attr"`
	PackagePrefix *string   `xml:"packagePrefix,attr,omitEmpty"`
}

type OrderEntry struct {
	XMLName xml.Name `xml:"orderEntry"`
	Type    string   `xml:"type,attr"`

	ForTests *bool `xml:"forTests,attr,omitEmpty"`

	LibraryName  *string `xml:"name,attr,omitEmpty"`
	LibraryLevel *string `xml:"level,attr,omitEmpty"`

	ModuleName *string `xml:"module-name,attr,omitEmpty"`
}

func NewLibraryEntry(name, level string) OrderEntry {
	return OrderEntry{
		Type:         "library",
		LibraryName:  &name,
		LibraryLevel: &level,
	}
}

func NewModuleEntry(name string) OrderEntry {
	return OrderEntry{
		Type:       "module",
		ModuleName: &name,
	}
}

func NewInheritedJdkEntry() OrderEntry {
	return OrderEntry{
		Type: "inheritedJdk",
	}
}

func NewSourceFolderEntry(forTests bool) OrderEntry {
	return OrderEntry{
		Type: "sourceFolder",
		ForTests: &forTests,
	}
}

func NewSourceFolder(url string, forTests bool, packagePrefix string) SourceFolder {
	return SourceFolder{
		Url:           url,
		ForTests:      forTests,
		PackagePrefix: &packagePrefix,
	}
}

func toModuleAndLibrary(graph *core.BuildGraph, buildTarget *core.BuildTarget) (Module,*Library) {
	for _, label := range buildTarget.PrefixedLabels("rule:") {
		if label == "java_library" {
			return NewJavaModule(graph, buildTarget), nil
		}
		if label == "java_test_library" {
			return NewJavaModule(graph, buildTarget), nil
		}
	}
	return NewWebModule(graph, buildTarget), nil
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
