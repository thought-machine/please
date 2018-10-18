package intellij

import (
	"core"
	"encoding/xml"
	"fmt"
	"io"
)

/*
<?xml version="1.0" encoding="UTF-8"?>
<project version="4">
  <component name="ProjectRootManager" version="2" languageLevel="JDK_10" default="true" project-jdk-name="10" project-jdk-type="JavaSDK">
    <output url="file://$PROJECT_DIR$/out" />
  </component>
</project>%
 */
type Misc struct {
	XMLName   xml.Name      `xml:"project"`
	Version   int           `xml:"version,attr"`
	Component MiscComponent `xml:"component"`
}

func NewMisc(javaSourceLevel int) Misc {
	return Misc{
		Version:   4,
		Component: NewMiscComponent(javaSourceLevel),
	}
}

func (misc *Misc) toXml(w io.Writer) {
	encoder := xml.NewEncoder(w)
	encoder.EncodeToken(xml.ProcInst{Target: "xml", Inst: []byte("version=\"1.0\" encoding=\"UTF-8\"")})
	encoder.Encode(misc)
}

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

func NewMiscComponent(javaSourceLevel int) MiscComponent {
	format := "JDK_%d"
	if javaSourceLevel < 10 {
		format = "JDK_1_%d"
	}
	return MiscComponent{
		Name:    "ProjectRootManager",
		Version: 2,
		Default: false,
		LanguageLevel: fmt.Sprintf(format, javaSourceLevel),
		ProjectJdkName: fmt.Sprintf("%d", javaSourceLevel),
		ProjectJdkType: "JavaSDK",
		Output:  NewMiscOutput(),
	}
}

type MiscOutput struct {
	XMLName xml.Name `xml:"output"`
	Url     string   `xml:"url,attr"`
}

func NewMiscOutput() MiscOutput {
	return MiscOutput{
		Url: "file://$PROJECT_DIR$/out",
	}
}

/*
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

type Modules struct {
	XMLName xml.Name `xml:"project"`
	Version int `xml:"version,attr"`
	Component ModulesComponent `xml:"component"`
}

func NewModules(targets core.BuildTargets) Modules {
	return Modules{
		Version: 4,
		Component: NewModulesComponent(targets),
	}
}

func (modules *Modules) toXml(w io.Writer) {
	encoder := xml.NewEncoder(w)
	encoder.EncodeToken(xml.ProcInst{Target: "xml", Inst: []byte("version=\"1.0\" encoding=\"UTF-8\"")})
	encoder.Encode(modules)
}

type ModulesComponent struct {
	XMLName xml.Name  `xml:"component"`
	Name string `xml:"name,attr"`
	Modules []ModulesModule `xml:"modules>module"`
}

func NewModulesComponent(targets core.BuildTargets) ModulesComponent {
	component := ModulesComponent{
		Name: "ProjectModuleManager",
	}

	for _, t := range targets {
		component.Modules = append(component.Modules, NewModulesModule(t))
	}

	return component
}

type ModulesModule struct {
	XMLName xml.Name `xml:"module"`
	FileUrl string `xml:"fileurl,attr"`
	FilePath string `xml:"filepath,attr"`
}

func NewModulesModule(target *core.BuildTarget) ModulesModule {
	filePath := "$PROJECT_DIR$/" + target.Label.PackageDir() + "/" + fmt.Sprintf("%s.iml", moduleName(target))
	return ModulesModule{
		FileUrl: "file://"+filePath,
		FilePath: filePath,
	}
}