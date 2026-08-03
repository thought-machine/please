package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCycleDetector(t *testing.T) {
	newTarget := func(state *BuildState, label string, deps ...string) *BuildTarget {
		target := NewBuildTarget(ParseBuildLabel(label, ""))
		for _, dep := range deps {
			target.AddDependency(ParseBuildLabel(dep, ""))
		}
		state.Graph.AddTarget(target)
		return target
	}

	t.Run("NoCycle", func(t *testing.T) {
		state := NewDefaultBuildState()
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
		state := NewDefaultBuildState()
		newTarget(state, "//src:a", "//src:b", "//src:c")
		newTarget(state, "//src:b", "//src:d", "//src:e")
		newTarget(state, "//src:c", "//src:b", "//src:f")
		newTarget(state, "//src:d", "//src:f")
		e := newTarget(state, "//src:e", "//src:f")
		f := newTarget(state, "//src:f", "//src:g")
		g := newTarget(state, "//src:g", "//src:e")

		detector := cycleDetector{graph: state.Graph}
		err := detector.Check()
		require.NotNil(t, err)
		require.Equal(t, 3, len(err.Cycle))
		log.Warning("%s", err)
		assert.Equal(t, []*BuildTarget{g, e, f}, err.Cycle)
	})
}
