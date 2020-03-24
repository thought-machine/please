package query

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

func TestQueryEntireGraph(t *testing.T) {
	graph := makeJSONGraph(makeGraph(), nil)
	assert.Equal(t, 2, len(graph.Packages))
	pkg1 := graph.Packages["package1"]
	assert.Equal(t, 2, len(pkg1.Targets))
	assert.Equal(t, 0, len(pkg1.Targets["target1"].Deps))
	assert.Equal(t, []string{"//package1:target1"}, pkg1.Targets["target2"].Deps)
	pkg2 := graph.Packages["package2"]
	assert.Equal(t, 1, len(pkg2.Targets))
	assert.Equal(t, []string{"//package1:target2"}, pkg2.Targets["target3"].Deps)
}

func TestQuerySingleTarget(t *testing.T) {
	graph := makeJSONGraph(makeGraph(), []core.BuildLabel{core.ParseBuildLabel("//package1:target2", "")})
	assert.Equal(t, 1, len(graph.Packages))
	pkg1 := graph.Packages["package1"]
	assert.Equal(t, 2, len(pkg1.Targets))
	assert.Equal(t, []string{"//package1:target1"}, pkg1.Targets["target2"].Deps)
}

func TestQueryPackage(t *testing.T) {
	graph := makeJSONGraph(makeGraph(), []core.BuildLabel{core.ParseBuildLabel("//package1:all", "")})
	assert.Equal(t, 1, len(graph.Packages))
	pkg1 := graph.Packages["package1"]
	assert.Equal(t, 2, len(pkg1.Targets))
	assert.Equal(t, 0, len(pkg1.Targets["target1"].Deps))
	assert.Equal(t, []string{"//package1:target1"}, pkg1.Targets["target2"].Deps)
}

func makeGraph() *core.BuildState {
	state := core.NewDefaultBuildState()
	graph := state.Graph
	pkg1 := core.NewPackage("package1")
	pkg1.AddTarget(makeTarget("//package1:target1"))
	pkg1.AddTarget(makeTarget("//package1:target2", "//package1:target1"))
	graph.AddPackage(pkg1)
	graph.AddTarget(pkg1.Target("target1"))
	graph.AddTarget(pkg1.Target("target2"))
	pkg2 := core.NewPackage("package2")
	pkg2.AddTarget(makeTarget("//package2:target3", "//package1:target2"))
	graph.AddPackage(pkg2)
	graph.AddTarget(pkg2.Target("target3"))
	graph.AddDependencySync(pkg1.Target("target2"), core.ParseBuildLabel("//package1:target1", ""))
	graph.AddDependencySync(pkg2.Target("target3"), core.ParseBuildLabel("//package1:target2", ""))
	return state
}

func makeTarget(label string, deps ...string) *core.BuildTarget {
	target := core.NewBuildTarget(core.ParseBuildLabel(label, ""))
	for _, dep := range deps {
		target.AddDependency(core.ParseBuildLabel(dep, ""))
	}
	return target
}
