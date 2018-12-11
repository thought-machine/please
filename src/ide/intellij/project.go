package intellij

import (
	"github.com/thought-machine/please/src/core"
	"encoding/xml"
	"fmt"
	"io"
)

// Misc is an ancillary structure that handles things like Java source level.
type Misc struct {
	XMLName   xml.Name      `xml:"project"`
	Version   int           `xml:"version,attr"`
	Component MiscComponent `xml:"component"`
}

func newMisc(javaSourceLevel int) Misc {
	return Misc{
		Version:   4,
		Component: newMiscComponent(javaSourceLevel),
	}
}

func (misc *Misc) toXML(w io.Writer) {
	w.Write([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"))
	content, err := xml.MarshalIndent(misc, "", "   ")

	if err  == nil {
		w.Write(content)
	}
}

// MiscComponent is the main Misc wrapper type.
type MiscComponent struct {
	XMLName        xml.Name   `xml:"component"`
	Name           string     `xml:"name,attr"`
	Version        int        `xml:"version,attr"`
	LanguageLevel  string     `xml:"languageLevel,attr"`
	Default        bool       `xml:"default,attr"`
	ProjectJdkName string     `xml:"project-jdk-name,attr"`
	ProjectJdkType string     `xml:"project-jdk-type,attr"`
	Output         MiscOutput `xml:"output"`
}

func newMiscComponent(javaSourceLevel int) MiscComponent {
	format := "JDK_%d"
	if javaSourceLevel < 10 {
		format = "JDK_1_%d"
	}
	return MiscComponent{
		Name:           "ProjectRootManager",
		Version:        2,
		Default:        false,
		LanguageLevel:  fmt.Sprintf(format, javaSourceLevel),
		ProjectJdkName: fmt.Sprintf("%d", javaSourceLevel),
		ProjectJdkType: "JavaSDK",
		Output:         newMiscOutput(),
	}
}

// MiscOutput determines where intellij puts the code it has compiled itself.
type MiscOutput struct {
	XMLName xml.Name `xml:"output"`
	URL     string   `xml:"url,attr"`
}

func newMiscOutput() MiscOutput {
	return MiscOutput{
		URL: "file://$PROJECT_DIR$/out",
	}
}

// Modules are the main structure that tells IntelliJ where to find all the modules it knows about.
type Modules struct {
	XMLName xml.Name `xml:"project"`
	Version int `xml:"version,attr"`
	Component ModulesComponent `xml:"component"`
}

func newModules(targets core.BuildTargets) Modules {
	return Modules{
		Version:   4,
		Component: newModulesComponent(targets),
	}
}

func (modules *Modules) toXML(w io.Writer) {
	w.Write([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"))
	content, err := xml.MarshalIndent(modules, "", "   ")

	if err  == nil {
		w.Write(content)
	}
}

// ModulesComponent represents all modules in the workspace.
type ModulesComponent struct {
	XMLName xml.Name  `xml:"component"`
	Name string `xml:"name,attr"`
	Modules []ModulesModule `xml:"modules>module"`
}

func newModulesComponent(targets core.BuildTargets) ModulesComponent {
	component := ModulesComponent{
		Name: "ProjectModuleManager",
	}

	for _, t := range targets {
		component.Modules = append(component.Modules, newModulesModule(t.Label))
	}

	return component
}

// ModulesModule represents one module in the workspace, and where to find its definition.
type ModulesModule struct {
	XMLName  xml.Name `xml:"module"`
	FileURL  string   `xml:"fileurl,attr"`
	FilePath string   `xml:"filepath,attr"`
}

func newModulesModule(label core.BuildLabel) ModulesModule {
	filePath := "$PROJECT_DIR$/" + label.PackageDir() + "/" + fmt.Sprintf("%s.iml", moduleName(label))
	return ModulesModule{
		FileURL:  "file://"+filePath,
		FilePath: filePath,
	}
}
