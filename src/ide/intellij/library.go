package intellij

import (
	"encoding/xml"
	"io"

	"github.com/thought-machine/please/src/core"
)

type libraryComponent struct {
	XMLName xml.Name `xml:"component"`
	Name string `xml:"name,attr"`
	Library Library `xml:"library"`
}

// Content is a simple wrapper for a URL.
type Content struct {
	XMLName    xml.Name `xml:"root"`
	ContentURL string   `xml:"url,attr"`
}

// Library represents an IntelliJ project library, which usually consists of a jar containing classes, but can
// also contain javadocs and sources.
type Library struct {
	XMLName      xml.Name  `xml:"library"`
	Name         string    `xml:"name,attr"`
	ClassPaths   []Content `xml:"CLASSES>root"`
	JavadocPaths []Content `xml:"JAVADOC>root"`
	SourcePaths  []Content `xml:"SOURCES>root"`
}

func newLibrary(graph *core.BuildGraph, target *core.BuildTarget) Library {
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
						ContentURL: "jar://$PROJECT_DIR$/../../" + depTarget.OutDir() + "/" + o + "!/",
					})
				}
			}
			if depTarget.HasLabel("maven-classes") {
				for _, o := range depTarget.Outputs() {
					classes = append(classes, Content{
						ContentURL: "jar://$PROJECT_DIR$/../../" + depTarget.OutDir() + "/" + o + "!/",
					})
				}

			}
			if depTarget.HasLabel("maven-javadocs") {
				for _, o := range depTarget.Outputs() {
					javadocs = append(javadocs, Content{
						ContentURL: "jar://$PROJECT_DIR$/../../" + depTarget.OutDir() + "/" + o + "!/",
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

func (library *Library) toXML(writer io.Writer) {
	table := libraryComponent{
		Name: "libraryTable",
		Library: *library,
	}
	contents, err := xml.MarshalIndent(table, "", "  ")
	if err == nil {
		writer.Write(contents)
	}
}
