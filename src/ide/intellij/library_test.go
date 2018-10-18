package intellij

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLibraryToXml(t *testing.T) {
	finagleBaseHttp := &Library{
		Name: "finagle-base-http",
		ClassPaths: []Content{
			{
				ContentUrl: "jar://$USER_HOME$/code/git.corp.tmachine.io/CORE/plz-out/gen/third_party/java/com/twitter/finagle-base-http.jar!/",
			},
		},
		SourcePaths: []Content{
			{
				ContentUrl: "jar://$USER_HOME$/code/git.corp.tmachine.io/CORE/plz-out/gen/third_party/java/com/twitter/finagle-base-http_src.jar!/",
			},
		},
	}

	buf := &bytes.Buffer{}
	finagleBaseHttp.toXml(buf)
	assert.Equal(t,
		"<?xml version=\"1.0\" encoding=\"UTF-8\"?>"+
			"<component name=\"libraryTable\">"+
			"<library name=\"finagle-base-http\">"+
			"<CLASSES>"+
			"<root contentUrl=\"jar://$USER_HOME$/code/git.corp.tmachine.io/CORE/plz-out/gen/third_party/java/com/twitter/finagle-base-http.jar!/\"></root>"+
			"</CLASSES>"+
			"<JAVADOC></JAVADOC>"+
			"<SOURCES>"+
			"<root contentUrl=\"jar://$USER_HOME$/code/git.corp.tmachine.io/CORE/plz-out/gen/third_party/java/com/twitter/finagle-base-http_src.jar!/\"></root>"+
			"</SOURCES>"+
			"</library>"+
			"</component>", buf.String())
}