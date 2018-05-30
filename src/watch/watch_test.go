package watch

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"core"
)

func TestCommands(t *testing.T) {
	state := core.NewBuildState(1, nil, 4, core.DefaultConfiguration())
	assert.Equal(t, []string{"build"}, commands(state, nil, false))
	assert.Equal(t, []string{"run"}, commands(state, nil, true))
	target := core.NewBuildTarget(core.NewBuildLabel("src/watch", "watch_test"))
	state.Graph.AddTarget(target)
	labels := []core.BuildLabel{target.Label}
	assert.Equal(t, []string{"build"}, commands(state, labels, false))
	target.IsTest = true
	assert.Equal(t, []string{"test"}, commands(state, labels, false))
	assert.Equal(t, []string{"run"}, commands(state, labels, true))
}
