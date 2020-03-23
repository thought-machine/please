package gc

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

func TestTargetsToRemoveWithTests(t *testing.T) {
	graph := createGraph()
	labels, _ := targetsToRemove(graph, nil, nil, nil, nil, true)
	assert.EqualValues(t, []core.BuildLabel{
		bl("//src/cli:cli"),
		bl("//src/parse:parse"),
	}, labels)
}

func TestTargetsToRemoveWithoutTests(t *testing.T) {
	graph := createGraph()
	labels, _ := targetsToRemove(graph, nil, nil, nil, nil, false)
	assert.EqualValues(t, []core.BuildLabel{
		bl("//src/cli:cli"),
		bl("//src/parse:parse"),
	}, labels)
}

func TestTargetsToRemoveWithArgs(t *testing.T) {
	graph := createGraph()
	labels, _ := targetsToRemove(graph, nil, []core.BuildLabel{bl("//src/cli:cli")}, nil, nil, false)
	assert.EqualValues(t, []core.BuildLabel{
		bl("//src/parse:parse"),
	}, labels)
}

func TestTargetsToRemoveFiltered(t *testing.T) {
	graph := createGraph()
	labels, _ := targetsToRemove(graph, []core.BuildLabel{bl("//src/cli:all")}, nil, nil, nil, false)
	assert.EqualValues(t, []core.BuildLabel{
		bl("//src/cli:cli"),
	}, labels)
}

func createGraph() *core.BuildGraph {
	graph := core.NewGraph()
	createTarget(graph, "//src/core:core")
	createTarget(graph, "//src/gc:gc", "//src/core:core")
	createTest(graph, "//src/core:core_test", "//src/core:core")
	createTarget(graph, "//src:please", "//src/core:core", "//src/gc:gc").IsBinary = true
	createTarget(graph, "//src/gc:test_lib", "//src/core:core").TestOnly = true
	createTest(graph, "//src/gc:gc_test", "//src/gc:test_lib", "//src/gc:gc")
	createTarget(graph, "//src/parse:parse", "//src/core:core")
	createTarget(graph, "//src/cli:cli")
	return graph
}

func createTarget(graph *core.BuildGraph, name string, deps ...string) *core.BuildTarget {
	target := core.NewBuildTarget(bl(name))
	graph.AddTarget(target)
	for _, dep := range deps {
		label := bl(dep)
		target.AddDependency(label)
		graph.AddDependency(target, label)
	}
	return target
}

func createTest(graph *core.BuildGraph, name string, deps ...string) *core.BuildTarget {
	target := createTarget(graph, name, deps...)
	target.IsBinary = true
	target.IsTest = true
	return target
}

func bl(in string) core.BuildLabel {
	return core.ParseBuildLabel(in, "")
}
