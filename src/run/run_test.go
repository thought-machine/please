package run

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/process"
)

func init() {
	if err := os.Chdir("src/run/test_data"); err != nil {
		panic(err)
	}
}

func TestSequential(t *testing.T) {
	state, labels1, labels2 := makeState(core.DefaultConfiguration())
	code := Sequential(state, labels1, nil, process.Quiet, false, false, false, "")
	assert.Equal(t, 0, code)
	code = Sequential(state, labels2, nil, process.Default, false, false, false, "")
	assert.Equal(t, 1, code)
}

func TestParallel(t *testing.T) {
	state, labels1, labels2 := makeState(core.DefaultConfiguration())
	code := Parallel(context.Background(), state, labels1, nil, 5, process.Default, false, false, false, false, "")
	assert.Equal(t, 0, code)
	code = Parallel(context.Background(), state, labels2, nil, 5, process.Quiet, false, false, false, false, "")
	assert.Equal(t, 1, code)
}

func TestEnvVars(t *testing.T) {
	config := core.DefaultConfiguration()
	config.Build.Path = []string{"/wibble"}
	state, lab1, _ := makeState(config)

	t.Setenv("PATH", "/usr/local/bin:/usr/bin:/bin")
	env := environ(state, state.Graph.TargetOrDie(lab1[0].BuildLabel), false, false)
	assert.Contains(t, env, "PATH=/usr/local/bin:/usr/bin:/bin")
	assert.NotContains(t, env, "PATH=/wibble")
	env = environ(state, state.Graph.TargetOrDie(lab1[0].BuildLabel), true, false)
	assert.NotContains(t, env, "PATH=/usr/local/bin:/usr/bin:/bin")
	assert.Contains(t, env, "PATH=:/wibble", env)
}

func makeState(config *core.Configuration) (*core.BuildState, []core.AnnotatedOutputLabel, []core.AnnotatedOutputLabel) {
	state := core.NewBuildState(config)
	target1 := core.NewBuildTarget(core.ParseBuildLabel("//:true", ""))
	target1.IsBinary = true
	target1.AddOutput("true")
	target1.Test = new(core.TestFields)
	state.Graph.AddTarget(target1)
	target2 := core.NewBuildTarget(core.ParseBuildLabel("//:false", ""))
	target2.IsBinary = true
	target2.AddOutput("false")
	target2.Test = new(core.TestFields)
	state.Graph.AddTarget(target2)
	return state, annotate([]core.BuildLabel{target1.Label}), annotate([]core.BuildLabel{target1.Label, target2.Label})
}

func annotate(labels []core.BuildLabel) []core.AnnotatedOutputLabel {
	ls := make([]core.AnnotatedOutputLabel, len(labels))
	for i, l := range labels {
		ls[i] = core.AnnotatedOutputLabel{
			BuildLabel: l,
		}
	}
	return ls
}
