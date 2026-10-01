package plz

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thought-machine/please/src/core"
)

func TestCycleDetector(t *testing.T) {
	newTarget := func(state *core.BuildState, label string, deps ...string) *core.BuildTarget {
		target := core.NewBuildTarget(core.ParseBuildLabel(label, ""))
		for _, dep := range deps {
			target.AddDependency(core.ParseBuildLabel(dep, ""))
		}
		state.Graph.AddTarget(target)
		return target
	}

	t.Run("NoCycle", func(t *testing.T) {
		state := core.NewDefaultBuildState()
		newTarget(state, "//src:a", "//src:b", "//src:c")
		newTarget(state, "//src:b", "//src:d", "//src:e")
		newTarget(state, "//src:c", "//src:b", "//src:f")
		newTarget(state, "//src:d", "//src:f")
		newTarget(state, "//src:e", "//src:f")
		newTarget(state, "//src:f", "//src:g")
		newTarget(state, "//src:g")

		detector := cycleDetector{graph: state.Graph}
		assert.Nil(t, detector.Check())
	})

	t.Run("Cycle", func(t *testing.T) {
		state := core.NewDefaultBuildState()
		newTarget(state, "//src:a", "//src:b", "//src:c")
		newTarget(state, "//src:b", "//src:d", "//src:e")
		newTarget(state, "//src:c", "//src:b", "//src:f")
		newTarget(state, "//src:d", "//src:f")
		newTarget(state, "//src:e", "//src:f")
		newTarget(state, "//src:f", "//src:g")
		newTarget(state, "//src:g", "//src:e")

		detector := cycleDetector{graph: state.Graph}
		err := detector.Check()
		require.Error(t, err)
		cerr, ok := errors.AsType[*errCycle](err)
		require.True(t, ok)
		assert.Equal(t, []core.BuildLabel{
			core.ParseBuildLabel("//src:g", ""),
			core.ParseBuildLabel("//src:e", ""),
			core.ParseBuildLabel("//src:f", ""),
		}, cerr.Cycle)
	})
}

func TestCycleDetectorSubincludes(t *testing.T) {
	newTarget := func(state *core.BuildState, label string) *core.BuildTarget {
		target := core.NewBuildTarget(core.ParseBuildLabel(label, ""))
		state.Graph.AddTarget(target)
		return target
	}

	t.Run("MutualSubincludes", func(t *testing.T) {
		// Neither package can finish parsing: each is waiting on a subinclude target that the other
		// hasn't defined yet. Neither package is in the graph, and neither target exists.
		state := core.NewDefaultBuildState()
		state.Graph.AddSubinclude(core.ParseBuildLabel("//a:all", ""), core.ParseBuildLabel("//b:defs", ""))
		state.Graph.AddSubinclude(core.ParseBuildLabel("//b:all", ""), core.ParseBuildLabel("//a:defs", ""))

		detector := cycleDetector{graph: state.Graph}
		err := detector.Check()
		require.Error(t, err)
		cerr, ok := errors.AsType[*errCycle](err)
		require.True(t, ok)
		assert.Len(t, cerr.Cycle, 4)
		assert.Contains(t, err.Error(), "parse of //a")
		assert.Contains(t, err.Error(), "//b:defs")
	})

	t.Run("LocalSubincludeIsNotACycle", func(t *testing.T) {
		// //a:defs is defined before //a subincludes //b:defs, so it already exists and anything
		// waiting on it can proceed without the rest of //a. That's legal and mustn't be a cycle.
		state := core.NewDefaultBuildState()
		newTarget(state, "//a:defs")
		state.Graph.AddSubinclude(core.ParseBuildLabel("//a:all", ""), core.ParseBuildLabel("//b:defs", ""))
		state.Graph.AddSubinclude(core.ParseBuildLabel("//b:all", ""), core.ParseBuildLabel("//a:defs", ""))

		detector := cycleDetector{graph: state.Graph}
		assert.Nil(t, detector.Check())
	})

	t.Run("NestedSubincludes", func(t *testing.T) {
		// The cycle runs through a subinclude that itself subincludes something.
		state := core.NewDefaultBuildState()
		state.Graph.AddSubinclude(core.ParseBuildLabel("//a:all", ""), core.ParseBuildLabel("//defs:outer", ""))
		state.Graph.AddSubinclude(core.ParseBuildLabel("//defs:outer", ""), core.ParseBuildLabel("//defs:inner", ""))
		state.Graph.AddSubinclude(core.ParseBuildLabel("//defs:all", ""), core.ParseBuildLabel("//a:defs", ""))

		detector := cycleDetector{graph: state.Graph}
		require.Error(t, detector.Check())
	})
}
