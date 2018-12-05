package query

import (
	"fmt"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

func makeTarget(g *core.BuildGraph, packageName string, labelName string, outputs []string) *core.BuildTarget {
	l := core.ParseBuildLabel(fmt.Sprintf("//%s:%s", packageName, labelName), "")
	t := core.NewBuildTarget(l)

	p := g.Package(packageName, "")
	if p == nil {
		p = core.NewPackage(packageName)
		g.AddPackage(p)
	}
	for _, out := range outputs {
		t.AddOutput(out)
		p.MustRegisterOutput(out, t)
	}
	p.AddTarget(t)
	g.AddTarget(t)
	return t
}

func TestConstructsMapFromGraph(t *testing.T) {
	graph := core.NewGraph()
	m := filesToLabelMap(graph)
	assert.Equal(t, 0, len(m))

	label := core.ParseBuildLabel("//package1:target1", "")
	makeTarget(graph, "package1", "target1", []string{"out1", "out2"})
	m = filesToLabelMap(graph)
	assert.Equal(t, 2, len(m))
	for _, l := range m {
		assert.Equal(t, label.String(), l.String())
	}
}

func TestMapKeysContainFullPathFromProjectRoot(t *testing.T) {
	graph := core.NewGraph()
	makeTarget(graph, "package1", "target1", []string{"out1", "out2"})
	makeTarget(graph, "package1", "target2", []string{"out3"})
	makeTarget(graph, "package2", "target1", []string{"out4"})
	m := filesToLabelMap(graph)
	label1 := core.ParseBuildLabel("//package1:target1", "")
	label2 := core.ParseBuildLabel("//package1:target2", "")
	label3 := core.ParseBuildLabel("//package2:target1", "")

	p1 := graph.Package("package1", "")
	p2 := graph.Package("package2", "")

	assert.Equal(t, m[path.Join(p1.Target("target1").OutDir(), "out1")].String(), label1.String())
	assert.Equal(t, m[path.Join(p1.Target("target1").OutDir(), "out2")].String(), label1.String())
	assert.Equal(t, m[path.Join(p1.Target("target2").OutDir(), "out3")].String(), label2.String())
	assert.Equal(t, m[path.Join(p2.Target("target1").OutDir(), "out4")].String(), label3.String())
}
