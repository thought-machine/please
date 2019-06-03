package plz

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

func TestLimiter(t *testing.T) {
	state := core.NewDefaultBuildState()
	config, err := core.ReadConfigFiles([]string{"src/plz/test_data/test.plzconfig"}, nil)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(config.Limit))
	l := newLimiter(config)
	t1 := core.NewBuildTarget(core.ParseBuildLabel("//src/plz:target1", ""))
	t1.AddLabel("java")
	t1.AddLabel("go")
	t2 := core.NewBuildTarget(core.ParseBuildLabel("//src/plz:target2", ""))
	t2.AddLabel("java")
	t3 := core.NewBuildTarget(core.ParseBuildLabel("//src/plz:target3", ""))
	t3.AddLabel("go")
	t4 := core.NewBuildTarget(core.ParseBuildLabel("//src/plz:target4", ""))
	t4.AddLabel("java")

	// Nothing running, so t1 can run.
	assert.True(t, l.ShouldRun(state, t1, core.Build))
	// t2 hasn't hit a limit yet
	assert.True(t, l.ShouldRun(state, t2, core.Test))
	// nor has t3 because it's not labelled java
	assert.True(t, l.ShouldRun(state, t3, core.Build))
	// t4 should though
	assert.False(t, l.ShouldRun(state, t4, core.Build))
	// now mark t1 as done
	l.Done(t1)
	// and t4 should now run
	assert.True(t, l.ShouldRun(state, t4, core.Build))
}
