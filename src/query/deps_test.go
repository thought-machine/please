package query

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

func TestQueryDepsHiddenTarget(t *testing.T) {
	state := core.NewDefaultBuildState()

	pkg := core.NewPackage("")

	target := core.NewBuildTarget(core.ParseBuildLabel("//:t1", ""))
	hiddenTarget := core.NewBuildTarget(core.ParseBuildLabel("//:_t2#test", ""))

	target.AddDependency(hiddenTarget.Label)

	state.AddTarget(pkg, hiddenTarget)
	state.AddTarget(pkg, target)

	target.ResolveDependencies(state.Graph)
	hiddenTarget.ResolveDependencies(state.Graph)

	var hiddenOut bytes.Buffer
	deps(&hiddenOut, state, []core.BuildLabel{target.Label}, true, -1)

	var publicOut bytes.Buffer
	deps(&publicOut, state, []core.BuildLabel{target.Label}, false, -1)

	expectedWithHidden := "//:t1\n  //:_t2#test\n"
	expectedWithoutHidden := "//:t1\n"

	assert.Equal(t, expectedWithHidden, hiddenOut.String())
	assert.Equal(t, expectedWithoutHidden, publicOut.String())
}

func TestQueryDepsInlcudeLabels(t *testing.T) {
	state := core.NewDefaultBuildState()

	pkg := core.NewPackage("")

	target := core.NewBuildTarget(core.ParseBuildLabel("//:t1", ""))
	fooTarget := core.NewBuildTarget(core.ParseBuildLabel("//:t2", ""))
	barTarget := core.NewBuildTarget(core.ParseBuildLabel("//:t3", ""))

	fooTarget.AddLabel("foo")

	target.AddDependency(fooTarget.Label)
	target.AddDependency(barTarget.Label)

	state.AddTarget(pkg, fooTarget)
	state.AddTarget(pkg, barTarget)
	state.AddTarget(pkg, target)

	target.ResolveDependencies(state.Graph)
	fooTarget.ResolveDependencies(state.Graph)
	barTarget.ResolveDependencies(state.Graph)

	var out bytes.Buffer
	deps(&out, state, []core.BuildLabel{target.Label}, false, -1)

	expected := "//:t1\n  //:t2\n  //:t3\n"
	assert.Equal(t, expected, out.String())

	state.Include = []string{"foo"}

	var out2 bytes.Buffer
	deps(&out2, state, []core.BuildLabel{target.Label}, false, -1)
	expected = "//:t2\n"
	assert.Equal(t, expected, out2.String())
}

func TestQueryDepsLevel(t *testing.T) {
	state := core.NewDefaultBuildState()

	pkg := core.NewPackage("")

	target := core.NewBuildTarget(core.ParseBuildLabel("//:t1", ""))
	fooTarget := core.NewBuildTarget(core.ParseBuildLabel("//:t2", ""))
	barTarget := core.NewBuildTarget(core.ParseBuildLabel("//:t3", ""))

	fooTarget.AddLabel("foo")

	target.AddDependency(fooTarget.Label)
	fooTarget.AddDependency(barTarget.Label)

	state.AddTarget(pkg, fooTarget)
	state.AddTarget(pkg, barTarget)
	state.AddTarget(pkg, target)

	target.ResolveDependencies(state.Graph)
	fooTarget.ResolveDependencies(state.Graph)
	barTarget.ResolveDependencies(state.Graph)

	var out bytes.Buffer
	deps(&out, state, []core.BuildLabel{target.Label}, false, -1)

	expected := "//:t1\n  //:t2\n    //:t3\n"
	assert.Equal(t, expected, out.String())

	var out2 bytes.Buffer
	deps(&out2, state, []core.BuildLabel{target.Label}, false, 1)
	expected = "//:t1\n  //:t2\n"
	assert.Equal(t, expected, out2.String())
}
