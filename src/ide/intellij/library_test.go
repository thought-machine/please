package intellij

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLibraryToXml(t *testing.T) {
	library := &Library{
		Name: "finagle-base-http",
		ClassPaths: []Content{
			{
				ContentURL: "jar://$USER_HOME$/code/git.corp.tmachine.io/CORE/plz-out/gen/third_party/java/com/twitter/finagle-base-http.jar!/",
			},
		},
		SourcePaths: []Content{
			{
				ContentURL: "jar://$USER_HOME$/code/git.corp.tmachine.io/CORE/plz-out/gen/third_party/java/com/twitter/finagle-base-http_src.jar!/",
			},
		},
	}

	buf := &bytes.Buffer{}
	library.toXML(buf)
	assert.Equal(t,
		"<component name=\"libraryTable\">\n"+
			"  <library name=\"finagle-base-http\">\n"+
			"    <CLASSES>\n"+
			"      <root url=\"jar://$USER_HOME$/code/git.corp.tmachine.io/CORE/plz-out/gen/third_party/java/com/twitter/finagle-base-http.jar!/\"></root>\n"+
			"    </CLASSES>\n"+
			"    <JAVADOC></JAVADOC>\n"+
			"    <SOURCES>\n"+
			"      <root url=\"jar://$USER_HOME$/code/git.corp.tmachine.io/CORE/plz-out/gen/third_party/java/com/twitter/finagle-base-http_src.jar!/\"></root>\n"+
			"    </SOURCES>\n"+
			"  </library>\n"+
			"</component>", buf.String())
}
