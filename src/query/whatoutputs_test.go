package query

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

func makeTarget(g *core.BuildGraph, label string, filegroup bool, outputs ...string) *core.BuildTarget {
	l := core.ParseBuildLabel(label, "")
	t := core.NewBuildTarget(l)

	p := g.Package(l.PackageName, "")
	if p == nil {
		p = core.NewPackage(l.PackageName)
		g.AddPackage(p)
	}
	if filegroup {
		t.IsFilegroup = true
		for _, out := range outputs {
			t.AddSource(core.FileLabel{File: out, Package: l.PackageName})
		}
	} else {
		for _, out := range outputs {
			t.AddOutput(out)
			p.MustRegisterOutput(out, t)
		}
	}
	p.AddTarget(t)
	g.AddTarget(t)
	return t
}

func TestDetectsOutputs(t *testing.T) {
	graph := core.NewGraph()
	makeTarget(graph, "//package1:target1", false, "out1", "out2")
	targets := whatOutputs(graph.AllTargets(), "plz-out/gen/package1/out1")
	assert.Equal(t, []core.BuildLabel{{PackageName: "package1", Name: "target1"}}, targets)
}

func TestDetectOutputsFilegroup(t *testing.T) {
	// Slightly different because filegroups register outputs in different ways and
	// there can be multiple outputting one file.
	graph := core.NewGraph()
	makeTarget(graph, "//package1:target1", true, "out1", "out2")
	makeTarget(graph, "//package1:target2", true, "out1")
	targets := whatOutputs(graph.AllTargets(), "plz-out/gen/package1/out1")
	assert.Equal(t, []core.BuildLabel{
		{PackageName: "package1", Name: "target1"},
		{PackageName: "package1", Name: "target2"},
	}, targets)
	targets = whatOutputs(graph.AllTargets(), "plz-out/gen/package1/out2")
	assert.Equal(t, []core.BuildLabel{{PackageName: "package1", Name: "target1"}}, targets)
}
