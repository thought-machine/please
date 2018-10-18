package intellij

import (
	"bytes"
	"fmt"
	"testing"

	"core"
)

func TestJavaModule(t *testing.T) {
	graph := core.NewGraph()

	subTarget := &core.BuildTarget{
		Label: core.BuildLabel{
			PackageName: "some/subpackage",
			Name:        "other_target",
		},
		Labels: []string{
			"rule:java_library",
			"package_prefix:com.mycompany.app1",
		},
	}

	graph.AddTarget(subTarget)

	target := &core.BuildTarget{
		Label: core.BuildLabel{
			PackageName: "some/package",
			Name:        "some_target",
		},
		Sources: []core.BuildInput{
			core.FileLabel{
				File:    "Foo.java",
				Package: "some/package",
			},
			core.FileLabel{
				File:    "Bar.java",
				Package: "some/package",
			},
			core.FileLabel{
				File:    "Baz.java",
				Package: "some/package",
			},
		},
		Labels: []string{
			"rule:java_library",
			"package_prefix:com.mycompany.app1",
		},
	}

	graph.AddTarget(target)

	graph.AddDependency(target.Label, subTarget.Label)


	write(graph, target)
	write(graph, subTarget)
}

func write(graph *core.BuildGraph, target *core.BuildTarget) {
	fmt.Println("writing to " + moduleFileLocation(target))

	m, _ := toModuleAndLibrary(graph, target)

	buf := &bytes.Buffer{}
	m.toXml(buf)
	fmt.Println(buf.String())
}