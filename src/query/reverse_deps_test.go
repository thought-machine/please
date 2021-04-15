package query

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

func TestReverseDeps(t *testing.T) {
	state := core.NewDefaultBuildState()
	graph := state.Graph

	root := core.NewBuildTarget(core.ParseBuildLabel("//package:root", ""))
	branch := core.NewBuildTarget(core.ParseBuildLabel("//package:branch", ""))
	leaf := core.NewBuildTarget(core.ParseBuildLabel("//package:leaf", ""))
	branch.AddDependency(root.Label)
	leaf.AddDependency(branch.Label)
	graph.AddTarget(root)
	graph.AddTarget(branch)
	graph.AddTarget(leaf)
	branch.ResolveDependencies(graph)
	leaf.ResolveDependencies(graph)

	pkg := core.NewPackage("package")
	graph.AddPackage(pkg)

	labels := revDepsLabels(state, []core.BuildLabel{branch.Label}, false, -1)
	assert.ElementsMatch(t, core.BuildLabels{leaf.Label}, labels)
	labels = revDepsLabels(state, []core.BuildLabel{root.Label}, false, -1)
	assert.ElementsMatch(t, core.BuildLabels{branch.Label, leaf.Label}, labels)
	labels = revDepsLabels(state, []core.BuildLabel{root.Label}, false, 1)
	assert.ElementsMatch(t, core.BuildLabels{branch.Label}, labels)
}

func TestReverseDepsWithHidden(t *testing.T) {
	state := core.NewDefaultBuildState()
	graph := state.Graph

	foo := core.NewBuildTarget(core.ParseBuildLabel("//package:foo", ""))
	fooInter1 := core.NewBuildTarget(core.ParseBuildLabel("//package:_foo#tag1", ""))
	fooInter2 := core.NewBuildTarget(core.ParseBuildLabel("//package:_foo#tag2", ""))
	bar := core.NewBuildTarget(core.ParseBuildLabel("//package:bar", ""))
	foo.AddDependency(fooInter2.Label)
	fooInter2.AddDependency(fooInter1.Label)
	fooInter1.AddDependency(bar.Label)
	graph.AddTarget(foo)
	graph.AddTarget(fooInter1)
	graph.AddTarget(fooInter2)
	graph.AddTarget(bar)

	pkg := core.NewPackage("package")
	graph.AddPackage(pkg)

	labels := revDepsLabels(state, []core.BuildLabel{bar.Label}, false, 1)
	assert.ElementsMatch(t, core.BuildLabels{foo.Label}, labels)

	labels = revDepsLabels(state, []core.BuildLabel{bar.Label}, true, 1)
	assert.ElementsMatch(t, core.BuildLabels{fooInter1.Label}, labels)

	labels = revDepsLabels(state, []core.BuildLabel{bar.Label}, true, 2)
	assert.ElementsMatch(t, core.BuildLabels{fooInter1.Label, fooInter2.Label}, labels)

	labels = revDepsLabels(state, []core.BuildLabel{bar.Label}, true, 3)
	assert.ElementsMatch(t, core.BuildLabels{fooInter1.Label, fooInter2.Label, foo.Label}, labels)
}

func revDepsLabels(state *core.BuildState, labels []core.BuildLabel, hidden bool, depth int) core.BuildLabels {
	ts := FindRevdeps(state, labels, hidden, depth)

	ret := make([]core.BuildLabel, 0, len(ts))
	for t := range ts {
		ret = append(ret, t.Label)
	}
	return ret
}
