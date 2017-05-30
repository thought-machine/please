package run

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"core"
)

func init() {
	if err := os.Chdir("src/run/test_data"); err != nil {
		panic(err)
	}
}

func TestSequential(t *testing.T) {
	graph, labels1, labels2 := makeGraph()
	code := Sequential(graph, labels1, nil)
	assert.Equal(t, 0, code)
	code = Sequential(graph, labels2, nil)
	assert.Equal(t, 1, code)
}

func TestParallel(t *testing.T) {
	graph, labels1, labels2 := makeGraph()
	code := Parallel(graph, labels1, nil)
	assert.Equal(t, 0, code)
	code = Parallel(graph, labels2, nil)
	assert.Equal(t, 1, code)
}

func makeGraph() (*core.BuildGraph, []core.BuildLabel, []core.BuildLabel) {
	state := core.NewBuildState(1, nil, 0, core.DefaultConfiguration())
	target1 := core.NewBuildTarget(core.ParseBuildLabel("//:true", ""))
	target1.IsBinary = true
	target1.AddOutput("true")
	state.Graph.AddTarget(target1)
	target2 := core.NewBuildTarget(core.ParseBuildLabel("//:false", ""))
	target2.IsBinary = true
	target2.AddOutput("false")
	state.Graph.AddTarget(target2)
	return state.Graph, []core.BuildLabel{target1.Label}, []core.BuildLabel{target1.Label, target2.Label}
}
