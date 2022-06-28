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

func TestStripHostArch(t *testing.T) {
	state := core.NewDefaultBuildState()
	parser := newParser()
	parser.interpreter = newInterpreter(state, parser)
	scope := parser.interpreter.scope.NewScope()

	require.Equal(t, pyString(""), stripHostArch(scope, pyString(""), "linux_amd64"))
	require.Equal(t, pyString("//path/to/package:name"), stripHostArch(scope, pyString("///linux_amd64//path/to/package:name"), "linux_amd64"))
	require.Equal(t, pyString("///go//path/to/package:name"), stripHostArch(scope, pyString("///go_linux_amd64//path/to/package:name"), "linux_amd64"))
	require.Equal(t, pyString("///go//path/to/package:name"), stripHostArch(scope, pyString("///go_linux_amd64//path/to/package:name"), "linux_amd64"))
	require.Equal(t, pyString("///go_darwin_amd64//path/to/package:name"), stripHostArch(scope, pyString("///go_darwin_amd64//path/to/package:name"), "linux_amd64"))

}
