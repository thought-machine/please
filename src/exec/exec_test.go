package exec

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/build"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/process"
)

func TestNoBinaryTargetNoOverrideCommand(t *testing.T) {
	target := core.NewBuildTarget(core.NewBuildLabel("pkg", "t"))

	err := exec(core.NewDefaultBuildState(), process.Default, target, ".", nil, nil, "", false, process.NoSandbox)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is not marked as binary")
}

func TestPrepareRuntimeDir(t *testing.T) {
	state := core.NewDefaultBuildState()

	target := core.NewBuildTarget(core.NewBuildLabel("pkg", "t"))
	target.BuildTimeout = 10 * time.Second
	target.Command = "echo 1 > file1"
	target.AddOutput("file1")
	state.Graph.AddTarget(target)

	build.Init(state)
	if err := build.StoreTargetMetadata(target, &core.BuildMetadata{}); err != nil {
		panic(err)
	}
	build.Build(0, state, target.Label, false)

	err := core.PrepareRuntimeDir(state, target, "plz-out/exec/pkg")
	assert.Nil(t, err)
	assert.True(t, fs.FileExists("plz-out/exec/pkg/file1"))
}

func TestCommandWithMultipleOutputs(t *testing.T) {
	target := core.NewBuildTarget(core.NewBuildLabel("pkg", "t"))
	target.AddOutput("my-out-1")
	target.AddOutput("my-out-2")

	cmd, err := resolveCmd(core.NewDefaultBuildState(), target, "", "runtime/dir", process.NoSandbox)
	assert.Empty(t, cmd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "it has 2 outputs")
}

func TestCommandMountNotSandboxed(t *testing.T) {
	core.MustFindRepoRoot()

	target := core.NewBuildTarget(core.NewBuildLabel("pkg", "t"))
	target.AddOutput("my-out")

	cmd, err := resolveCmd(core.NewDefaultBuildState(), target, "", "runtime/dir", process.NoSandbox)
	assert.Nil(t, err)
	assert.Equal(t, filepath.Join(core.RepoRoot, "runtime/dir/my-out"), cmd)
}

func TestCommandMountSandboxed(t *testing.T) {
	target := core.NewBuildTarget(core.NewBuildLabel("pkg", "t"))
	target.AddOutput("my-out")

	cmd, err := resolveCmd(core.NewDefaultBuildState(), target, "", "runtime/dir", process.NewSandboxConfig(false, true))
	assert.Nil(t, err)
	assert.Equal(t, filepath.Join(core.SandboxDir, "my-out"), cmd)
}

func TestConvertEnv(t *testing.T) {
	env := ConvertEnv(map[string]string{
		"B": "1",
		"A": "2",
	})
	assert.Equal(t, []string{"A=2", "B=1"}, env)
}
