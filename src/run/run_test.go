package run

import (
	"context"
	"github.com/thought-machine/please/src/cli"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

func init() {
	if err := os.Chdir("src/run/test_data"); err != nil {
		panic(err)
	}
}

func TestSequential(t *testing.T) {
	state, labels1, labels2 := makeState(core.DefaultConfiguration())
	code := Sequential(state, labels1, nil, true, false, false, "", cli.Arch{})
	assert.Equal(t, 0, code)
	code = Sequential(state, labels2, nil, false, false, false, "", cli.Arch{})
	assert.Equal(t, 1, code)
}

func TestParallel(t *testing.T) {
	state, labels1, labels2 := makeState(core.DefaultConfiguration())
	code := Parallel(context.Background(), state, labels1, nil, 5, false, false, false, false, "", cli.Arch{})
	assert.Equal(t, 0, code)
	code = Parallel(context.Background(), state, labels2, nil, 5, true, false, false, false, "", cli.Arch{})
	assert.Equal(t, 1, code)
}

func TestEnvVars(t *testing.T) {
	config := core.DefaultConfiguration()
	config.Build.Path = []string{"/wibble"}
	state, lab1, _ := makeState(config)

	os.Setenv("PATH", "/usr/local/bin:/usr/bin:/bin")
	env := environ(state, state.Graph.TargetOrDie(lab1[0]), false)
	assert.Contains(t, env, "PATH=/usr/local/bin:/usr/bin:/bin")
	assert.NotContains(t, env, "PATH=/wibble")
	env = environ(state, state.Graph.TargetOrDie(lab1[0]), true)
	assert.NotContains(t, env, "PATH=/usr/local/bin:/usr/bin:/bin")
	assert.Contains(t, env, "PATH=:/wibble", env)
}

func makeState(config *core.Configuration) (*core.BuildState, []core.BuildLabel, []core.BuildLabel) {
	state := core.NewBuildState(config)
	target1 := core.NewBuildTarget(core.ParseBuildLabel("//:true", ""))
	target1.IsBinary = true
	target1.AddOutput("true")
	state.Graph.AddTarget(target1)
	target2 := core.NewBuildTarget(core.ParseBuildLabel("//:false", ""))
	target2.IsBinary = true
	target2.AddOutput("false")
	state.Graph.AddTarget(target2)
	return state, []core.BuildLabel{target1.Label}, []core.BuildLabel{target1.Label, target2.Label}
}
