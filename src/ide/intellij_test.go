package ide

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"core"
)

func TestNotModule(t *testing.T) {
	assert.Equal(t, nil, toModule(nil, &core.BuildTarget{}))
}

func TestJavaModule(t *testing.T) {

	prefix := "com.mycompany.app1"

	j := &JavaModule{
		Module: Module{
			contentUrl:   "file://$MODULE_DIR$/../../../../../../../src/app1/com/mycompany/app1",
			isTestSource: false,
		},
		packagePrefix: &prefix,
	}

	target := &core.BuildTarget{
		Sources: []core.BuildInput{
			core.FileLabel{
				File:    "Foo.java",
				Package: "src/app1/com/mycompany/app1",
			},
			core.FileLabel{
				File:    "Bar.java",
				Package: "src/app1/com/mycompany/app1",
			},
			core.FileLabel{
				File:    "Baz.java",
				Package: "src/app1/com/mycompany/app1",
			},
		},
		Labels: []string{
			"rule:java_library",
			"package_prefix:com.mycompany.app1",
		},
	}

	m := toModule(nil, target)

	assert.Equal(t, j, m, "Module and target didn't match")
}

func TestPathFromModuleFileToInputsSimple(t *testing.T) {

	inputs := []core.BuildInput{
		core.FileLabel{
			File:    "File1.java",
			Package: "some/simple/path",
		},
		core.FileLabel{
			File:    "File2.java",
			Package: "some/simple/path",
		},
	}

	expected := "some/simple/path"
	assert.Equal(t, expected, *commonDirectoryFromInputs(nil, inputs))
}

func TestPathFromModuleFileToInputsComplex(t *testing.T) {

	inputs := []core.BuildInput{
		core.FileLabel{
			File:    "File1.java",
			Package: "some/not_so_simple/path",
		},
		core.FileLabel{
			File:    "File2.java",
			Package: "some/simple/path",
		},
	}

	expected := "some"
	assert.Equal(t, expected, *commonDirectoryFromInputs(nil, inputs))
}

func TestPathFromModuleFileToInputsNoMatch(t *testing.T) {

	inputs := []core.BuildInput{
		core.FileLabel{
			File:    "File1.java",
			Package: "some/simple/path",
		},
		core.FileLabel{
			File:    "File2.java",
			Package: "another/simple/path",
		},
	}

	expected := "."
	assert.Equal(t, expected, *commonDirectoryFromInputs(nil, inputs))
}

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
