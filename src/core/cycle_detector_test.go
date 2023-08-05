package core

import (
	"testing"
	"time"

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
		go state.Build(target, ParseModeNormal)
		return target
	}

	waitForDeps := func(state *BuildState) {
		// Wait for all targets to have resolved all their dependencies.
		allDepsResolved := func() bool {
			for _, target := range state.Graph.AllTargets() {
				if len(target.DeclaredDependencies()) != len(target.Dependencies()) {
					return false
				}
			}
			return true
		}
		for i := 0; i < 1000; i++ {
			if allDepsResolved() {
				return
			}
			time.Sleep(2 * time.Millisecond)
		}
		panic("not all dependencies resolved")
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
		waitForDeps(state)

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
		waitForDeps(state)

		detector := cycleDetector{graph: state.Graph}
		err := detector.Check()
		require.NotNil(t, err)
		require.Equal(t, 3, len(err.Cycle))
		log.Warning("%s", err)
		assert.Equal(t, []*BuildTarget{g, e, f}, err.Cycle)
	})
}
