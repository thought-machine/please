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

	err := exec(core.NewDefaultBuildState(), process.Default, target, ".", nil, nil, nil, "", false, process.NoSandbox)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "target needs to be a binary")
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

func TestSimpleOverrideCommand(t *testing.T) {
	target := core.NewBuildTarget(core.NewBuildLabel("pkg", "t"))

	cmd, err := resolveCmd(core.NewDefaultBuildState(), target, []string{"ls", "-l"}, "", "runtime/dir", process.NoSandbox)
	assert.Nil(t, err)
	assert.Equal(t, "ls -l", cmd)
}

func TestOverrideCommandWithSequence(t *testing.T) {
	target := core.NewBuildTarget(core.NewBuildLabel("pkg", "t"))
	target.AddOutput("my-binary")
	target.IsBinary = true

	cmd, err := resolveCmd(core.NewDefaultBuildState(), target, []string{"$(out_exe", "//pkg:t)"}, "", "runtime/dir", process.NoSandbox)
	assert.Nil(t, err)
	assert.Equal(t, "plz-out/bin/pkg/my-binary", cmd)
}

func TestCommandWithMultipleOutputs(t *testing.T) {
	target := core.NewBuildTarget(core.NewBuildLabel("pkg", "t"))
	target.AddOutput("my-out-1")
	target.AddOutput("my-out-2")

	cmd, err := resolveCmd(core.NewDefaultBuildState(), target, nil, "", "runtime/dir", process.NoSandbox)
	assert.Empty(t, cmd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "it has 2 outputs")
}

func TestCommandMountNotSandboxed(t *testing.T) {
	core.MustFindRepoRoot()

	target := core.NewBuildTarget(core.NewBuildLabel("pkg", "t"))
	target.AddOutput("my-out")

	cmd, err := resolveCmd(core.NewDefaultBuildState(), target, nil, "", "runtime/dir", process.NoSandbox)
	assert.Nil(t, err)
	assert.Equal(t, filepath.Join(core.RepoRoot, "runtime/dir/my-out"), cmd)
}

func TestCommandMountSandboxed(t *testing.T) {
	target := core.NewBuildTarget(core.NewBuildLabel("pkg", "t"))
	target.AddOutput("my-out")

	cmd, err := resolveCmd(core.NewDefaultBuildState(), target, nil, "", "runtime/dir", process.NewSandboxConfig(false, true))
	assert.Nil(t, err)
	assert.Equal(t, filepath.Join(core.SandboxDir, "my-out"), cmd)
}

func TestExec(t *testing.T) {
	state := core.NewDefaultBuildState()
	target := core.NewBuildTarget(core.NewBuildLabel("pkg", "t"))
	state.Graph.AddTarget(target)

	err := exec(state, process.Default, target, "test", nil, []string{"echo", "foo"}, nil, "", false, process.NoSandbox)
	assert.Nil(t, err)
}

func TestCommandExitCode(t *testing.T) {
	state := core.NewDefaultBuildState()
	target := core.NewBuildTarget(core.NewBuildLabel("pkg", "t"))
	state.Graph.AddTarget(target)

	exitCode := Exec(state, core.AnnotatedOutputLabel{BuildLabel: target.Label}, "test", nil, []string{"exit", "5"}, false, process.NoSandbox)
	assert.Equal(t, 5, exitCode)
}
