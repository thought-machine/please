package intellij

import (
	"encoding/xml"
	"io"

	"core"
)

type libraryComponent struct {
	XMLName xml.Name `xml:"component"`
	Name string `xml:"name,attr"`
	Library Library `xml:"library"`
}

type Content struct {
	XMLName    xml.Name `xml:"root"`
	ContentUrl string   `xml:"url,attr"`
}

type Library struct {
	XMLName      xml.Name  `xml:"library"`
	Name         string    `xml:"name,attr"`
	ClassPaths   []Content `xml:"CLASSES>root"`
	JavadocPaths []Content `xml:"JAVADOC>root"`
	SourcePaths  []Content `xml:"SOURCES>root"`
}

func NewLibrary(graph *core.BuildGraph, target *core.BuildTarget) Library {
	classes := []Content{}
	javadocs := []Content{}
	sources := []Content{}
	for _, dep  := range target.Sources {
		label := dep.Label()
		if label != nil {
			depTarget := graph.TargetOrDie(*label)

			if depTarget.HasLabel("maven-sources") {
				for _, o := range depTarget.Outputs() {
					sources = append(sources, Content{
						ContentUrl: "jar://$PROJECT_DIR$/../../" + depTarget.OutDir() + "/" + o + "!/",
					})
				}
			}
			if depTarget.HasLabel("maven-classes") {
				for _, o := range depTarget.Outputs() {
					classes = append(classes, Content{
						ContentUrl: "jar://$PROJECT_DIR$/../../" + depTarget.OutDir() + "/" + o + "!/",
					})
				}

			}
			if depTarget.HasLabel("maven-javadocs") {
				for _, o := range depTarget.Outputs() {
					javadocs = append(javadocs, Content{
						ContentUrl: "jar://$PROJECT_DIR$/../../" + depTarget.OutDir() + "/" + o + "!/",
					})
				}

			}
		}
	}

	library := Library{
		Name: libraryName(target),
		SourcePaths: sources,
		ClassPaths: classes,
		JavadocPaths: javadocs,
	}

	return library
}

func (library *Library) toXml(writer io.Writer) {
	table := libraryComponent{
		Name: "libraryTable",
		Library: *library,
	}
	contents, err := xml.MarshalIndent(table, "", "  ")
	if err == nil {
		writer.Write(contents)
	}
}