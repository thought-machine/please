package intellij

import (
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"

	"github.com/thought-machine/please/src/core"
)

// Module represents the IntelliJ concept of a module
type Module struct {
	XMLName    xml.Name          `xml:"module"`
	ModuleType string            `xml:"type,attr"`
	Version    int               `xml:"version,attr"`
	Component  ModuleComponent   `xml:"component"`
}

func newModule(graph *core.BuildGraph, target *core.BuildTarget) Module {
	component := newModuleComponent(target)
	component.addOrderEntry(newSourceFolderEntry(false))

	for _, label := range target.DeclaredDependencies() {
		dep := graph.TargetOrDie(label)
		if shouldMakeModule(dep) {
			component.addOrderEntry(newModuleEntry(moduleName(label)))
		}
	}

	module := Module{
		ModuleType: "WEB_MODULE",
		Version:    4,
		Component: component,
	}

	return module
}

func newJavaModule(graph *core.BuildGraph, target *core.BuildTarget) Module {
	module := newModule(graph, target)
	module.ModuleType = "JAVA_MODULE"
	module.Component.addOrderEntry(newInheritedJdkEntry())

	if shouldMakeLibrary(target) {
		module.Component.addOrderEntry(newLibraryEntry(libraryName(target), "project"))
	}
	return module
}

func newScalaModule(graph *core.BuildGraph, target *core.BuildTarget) Module {
	module := newJavaModule(graph, target)

	module.Component.addOrderEntry(newLibraryEntry("scala-sdk", "application"))

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

func shouldMakeStaticModule(target *core.BuildTarget) bool {
	for _, label := range target.PrefixedLabels("rule:") {
		if label == "resources" || label == "test_resources" {
			return true
		}
	}
	return false
}

func shouldMakeModule(target *core.BuildTarget) bool {
	return shouldMakeStaticModule(target) ||
		shouldMakeJavaModule(target) ||
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
		} else if label == "resources" {
			return true
		} else if label == "test_resources" {
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

func (module *Module) toXML(writer io.Writer) {
	writer.Write([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"))

	content, err := xml.MarshalIndent(module, "", "  ")
	if err  == nil {
		writer.Write(content)
	}
}

// ModuleComponent represents the top level module wrapper.
type ModuleComponent struct {
	XMLName               xml.Name       `xml:"component"`
	Name                  string         `xml:"name,attr"`
	InheritCompilerOutput bool           `xml:"inherit-compiler-output,attr"`
	Content               *ModuleContent `xml:"content,omitEmpty"`
	OrderEntries          []OrderEntry   `xml:"orderEntry"`
}

func newModuleComponent(target *core.BuildTarget) ModuleComponent {
	orderEntries := []OrderEntry{}

	return ModuleComponent{
		Name: "NewModuleRootManager",
		InheritCompilerOutput: true,
		Content:               newModuleContent(target),
		OrderEntries:          orderEntries,
	}
}

func (moduleComponent *ModuleComponent) addOrderEntry(entry OrderEntry) {
	moduleComponent.OrderEntries = append(moduleComponent.OrderEntries, entry)
}

// ModuleContent is a wrapper that is generally only used once in a given module.
type ModuleContent struct {
	XMLName      xml.Name       `xml:"content"`
	URL          string         `xml:"url,attr"`
	SourceFolder []SourceFolder `xml:"sourceFolder"`
}

func newModuleContent(target *core.BuildTarget) *ModuleContent {
	if !shouldHaveContent(target) {
		return nil
	}

	sourceFolders := []SourceFolder{}
	commonDir := target.Label.PackageDir()

	joined := filepath.Join(core.RepoRoot, commonDir)
	location := relativisedPathTo(moduleDirLocation(target), &joined)

	sourceFolders = append(sourceFolders, SourceFolder{
		URL:           fmt.Sprintf("file://$MODULE_DIR$/%s", *location),
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
		URL:          fmt.Sprintf("file://$MODULE_DIR$/%s", *contentDir),
		SourceFolder: sourceFolders,
	}
}

// SourceFolder is a location on disk that contains files of interest to this package.
type SourceFolder struct {
	XMLName       xml.Name `xml:"sourceFolder"`
	URL           string   `xml:"url,attr"`
	IsTestSource  bool     `xml:"isTestSource,attr"`
	PackagePrefix *string  `xml:"packagePrefix,attr,omitEmpty"`
}

// OrderEntry represents a dependency on (e.g.) another module, or a library or an SDK.
type OrderEntry struct {
	XMLName xml.Name `xml:"orderEntry"`
	Type    string   `xml:"type,attr"`

	ForTests *bool `xml:"forTests,attr,omitEmpty"`

	Exported *string `xml:"exported,attr,omitEmpty"`

	LibraryName  *string `xml:"name,attr,omitEmpty"`
	LibraryLevel *string `xml:"level,attr,omitEmpty"`

	ModuleName *string `xml:"module-name,attr,omitEmpty"`
}

func newLibraryEntry(name, level string) OrderEntry {
	exported := ""
	return OrderEntry{
		Type:         "library",
		LibraryName:  &name,
		LibraryLevel: &level,
		Exported: &exported,
	}
}

func newModuleEntry(name string) OrderEntry {
	exported := ""
	return OrderEntry{
		Type:       "module",
		ModuleName: &name,
		Exported: &exported,
	}
}

func newInheritedJdkEntry() OrderEntry {
	return OrderEntry{
		Type: "inheritedJdk",
	}
}

func newSourceFolderEntry(forTests bool) OrderEntry {
	return OrderEntry{
		Type:     "sourceFolder",
		ForTests: &forTests,
	}
}

func toModule(graph *core.BuildGraph, buildTarget *core.BuildTarget) *Module {
	var module *Module

	if shouldMakeStaticModule(buildTarget) {
		madeModule := newModule(graph, buildTarget)
		module = &madeModule
	}

	if shouldMakeJavaModule(buildTarget) {
		madeModule := newJavaModule(graph, buildTarget)
		module = &madeModule
	}

	if shouldMakeScalaModule(buildTarget) {
		madeModule := newScalaModule(graph, buildTarget)
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
