package run

import (
	"context"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/process"
)

func init() {
	if err := os.Chdir("src/run/test_data"); err != nil {
		panic(err)
	}
}

// runnable returns the fixture name that this platform can actually execute.
//
// The Unix fixtures are shell scripts relying on a #! line, and Windows has no such mechanism:
// it decides what is executable by extension. The .cmd files beside them are the same two
// programs written the only way Windows will run one by name - which is exactly what the shell
// plugin does for an sh_binary there.
func runnable(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".cmd"
	}
	return name
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

	// Built rather than written out: the separator between entries differs per platform, and
	// so does what Please prepends - its own location, which is empty in this state.
	sep := string(os.PathListSeparator)
	hostPath := strings.Join([]string{"/usr/local/bin", "/usr/bin", "/bin"}, sep)

	t.Setenv("PATH", hostPath)
	env := environ(state, state.Graph.TargetOrDie(lab1[0].BuildLabel), false, false)
	assert.Equal(t, hostPath, envValue(t, env, "PATH"))
	env = environ(state, state.Graph.TargetOrDie(lab1[0].BuildLabel), true, false)
	assert.Equal(t, sep+"/wibble", envValue(t, env, "PATH"))
}

// envValue returns the value of one variable, and asserts there is exactly one entry for it.
//
// Looked up by name rather than matched as a whole string, because the OS decides how the name
// is spelled: Windows stores PATH as Path, so asserting on the literal "PATH=" finds nothing.
// The count is the point of the second assertion - appending a second entry instead of
// replacing the first is the bug this guards, and os/exec hides it by deduplicating.
func envValue(t *testing.T, env []string, name string) string {
	t.Helper()
	var values []string
	for _, entry := range env {
		if k, v, ok := strings.Cut(entry, "="); ok && envNamesEqual(k, name) {
			values = append(values, v)
		}
	}
	require.Len(t, values, 1, "expected exactly one %s in %v", name, env)
	return values[0]
}

func makeState(config *core.Configuration) (*core.BuildState, []core.AnnotatedOutputLabel, []core.AnnotatedOutputLabel) {
	state := core.NewBuildState(config)
	target1 := core.NewBuildTarget(core.ParseBuildLabel("//:true", ""))
	target1.IsBinary = true
	target1.AddOutput(runnable("true"))
	target1.Test = new(core.TestFields)
	state.Graph.AddTarget(target1)
	target2 := core.NewBuildTarget(core.ParseBuildLabel("//:false", ""))
	target2.IsBinary = true
	target2.AddOutput(runnable("false"))
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
