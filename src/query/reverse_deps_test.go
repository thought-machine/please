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
	graph.AddDependencySync(branch, root.Label)
	graph.AddDependencySync(leaf, branch.Label)

	pkg := core.NewPackage("package")
	pkg.AddTarget(root)
	pkg.AddTarget(branch)
	pkg.AddTarget(leaf)
	graph.AddPackage(pkg)

	labels := getRevDepTransitiveLabels(state, []core.BuildLabel{branch.Label}, map[core.BuildLabel]struct{}{}, -1)
	assert.Equal(t, core.BuildLabels{leaf.Label}, labels)
	labels = getRevDepTransitiveLabels(state, []core.BuildLabel{root.Label}, map[core.BuildLabel]struct{}{}, -1)
	assert.Equal(t, core.BuildLabels{branch.Label, leaf.Label}, labels)
	labels = getRevDepTransitiveLabels(state, []core.BuildLabel{root.Label}, map[core.BuildLabel]struct{}{}, 1)
	assert.Equal(t, core.BuildLabels{branch.Label}, labels)
}
