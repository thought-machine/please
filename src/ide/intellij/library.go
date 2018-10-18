package intellij

import (
	"encoding/xml"
	"io"
)

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