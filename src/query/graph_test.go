package query

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

func TestQueryEntireGraph(t *testing.T) {
	state := makeGraph(t)
	graph := makeJSONGraph(state, nil)
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
	graph := makeJSONGraph(makeGraph(t), []core.BuildLabel{core.ParseBuildLabel("//package1:target2", "")})
	assert.Equal(t, 1, len(graph.Packages))
	pkg1 := graph.Packages["package1"]
	assert.Equal(t, 2, len(pkg1.Targets))
	assert.Equal(t, []string{"//package1:target1"}, pkg1.Targets["target2"].Deps)
}

func TestQueryPackage(t *testing.T) {
	graph := makeJSONGraph(makeGraph(t), []core.BuildLabel{core.ParseBuildLabel("//package1:all", "")})
	assert.Equal(t, 1, len(graph.Packages))
	pkg1 := graph.Packages["package1"]
	assert.Equal(t, 2, len(pkg1.Targets))
	assert.Equal(t, 0, len(pkg1.Targets["target1"].Deps))
	assert.Equal(t, []string{"//package1:target1"}, pkg1.Targets["target2"].Deps)
}

func TestStreamedGraphMatchesInMemoryGraph(t *testing.T) {
	state := makeGraph(t)
	// Add a couple of subrepos (in non-sorted order) to check they get streamed correctly too.
	for _, subrepo := range []string{"zzz", "aaa"} {
		pkg := core.NewPackage("package3", core.WithPackageSubrepo(subrepo))
		target := makeTarget("///" + subrepo + "//package3:target4")
		pkg.AddTarget(target)
		state.Graph.AddTarget(target)
		state.Graph.AddPackage(pkg)
	}

	var streamed bytes.Buffer
	assert.NoError(t, writeGraph(&streamed, state, nil))

	var expected bytes.Buffer
	assert.NoError(t, json.MarshalEncode(jsontext.NewEncoder(&expected, graphEncodeOptions), makeJSONGraph(state, nil)))
	assert.Equal(t, expected.String(), streamed.String())

	// Sanity check the structure
	graph := &JSONGraph{}
	assert.NoError(t, json.Unmarshal(streamed.Bytes(), graph))
	assert.Equal(t, 2, len(graph.Packages))
	assert.Equal(t, 2, len(graph.Subrepos))
	assert.Contains(t, graph.Subrepos["aaa"].Packages["package3"].Targets, "target4")
	assert.Contains(t, graph.Subrepos["zzz"].Packages["package3"].Targets, "target4")
}

func TestStreamedGraphNoSubrepos(t *testing.T) {
	var streamed bytes.Buffer
	assert.NoError(t, writeGraph(&streamed, makeGraph(t), nil))
	assert.NotContains(t, streamed.String(), "subrepos")
	assert.True(t, jsontext.Value(streamed.Bytes()).IsValid())
}

func TestOrderedParallel(t *testing.T) {
	in := make([]int, 1000)
	for i := range in {
		in[i] = i
	}
	out := slices.Collect(orderedParallel(in, 8, func(i int) int { return i * 2 }))
	assert.Equal(t, len(in), len(out))
	for i, x := range out {
		assert.Equal(t, i*2, x)
	}
	// Stopping early must not deadlock.
	for x := range orderedParallel(in, 8, func(i int) int { return i }) {
		if x == 10 {
			break
		}
	}
}

func makeGraph(t *testing.T) *core.BuildState {
	t.Helper()
	state := core.NewDefaultBuildState()
	graph := state.Graph
	pkg1 := core.NewPackage("package1")
	pkg1.AddTarget(makeTarget("//package1:target1"))
	t2 := makeTarget("//package1:target2", "//package1:target1")
	pkg1.AddTarget(t2)
	graph.AddTarget(pkg1.Target("target1"))
	graph.AddTarget(pkg1.Target("target2"))
	graph.AddPackage(pkg1)
	pkg2 := core.NewPackage("package2")
	t3 := makeTarget("//package2:target3", "//package1:target2")
	pkg2.AddTarget(t3)
	graph.AddTarget(pkg2.Target("target3"))
	graph.AddPackage(pkg2)
	return state
}

func makeTarget(label string, deps ...string) *core.BuildTarget {
	target := core.NewBuildTarget(core.ParseBuildLabel(label, ""))
	for _, dep := range deps {
		target.AddDependency(core.ParseBuildLabel(dep, ""))
	}
	return target
}
