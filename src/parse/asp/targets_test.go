package asp

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/thought-machine/please/src/core"
)

func TestValidateTargetNoSandbox(t *testing.T) {
	state := core.NewDefaultBuildState()

	foo := core.NewBuildTarget(core.NewBuildLabel("pkg", "foo"))
	foo.Sandbox = false

	err := validateSandbox(state, foo)
	require.NoError(t, err)

	state.Config.Sandbox.ExcludeableTargets = []core.BuildLabel{core.NewBuildLabel("third_party", "all")}
	err = validateSandbox(state, foo)
	require.Error(t, err)

	state.Config.Sandbox.ExcludeableTargets = []core.BuildLabel{core.NewBuildLabel("pkg", "all")}
	err = validateSandbox(state, foo)
	require.NoError(t, err)
}

func TestValidateTargetSandbox(t *testing.T) {
	state := core.NewDefaultBuildState()
	state.Config.Sandbox.ExcludeableTargets = []core.BuildLabel{core.NewBuildLabel("third_party", "all")}

	foo := core.NewBuildTarget(core.NewBuildLabel("pkg", "foo"))
	foo.Sandbox = true

	err := validateSandbox(state, foo)
	require.NoError(t, err)

	foo.Test = new(core.TestFields)

	err = validateSandbox(state, foo)
	require.Error(t, err)

	foo.Test.Sandbox = true
	err = validateSandbox(state, foo)
	require.NoError(t, err)
}
