package command

import (
	"testing"

	"github.com/thought-machine/please/src/core"

	"github.com/stretchr/testify/assert"
)

func TestLocationExpansion(t *testing.T) {
	pkg := "foo/bar"

	cmd := parse("echo $(basename $(location :bar)) > $OUT", pkg)
	state := core.NewDefaultBuildState()


	bar := core.NewBuildTarget(core.ParseBuildLabel(":bar", pkg))
	bar.AddOutput("bar_out")
	state.AddTarget(core.NewPackage(pkg), bar)

	barTest := core.NewBuildTarget(core.ParseBuildLabel(":bar_test", pkg))
	barTest.AddDependency(bar.Label)
	state.AddTarget(core.NewPackage(pkg), barTest)

	assert.Equal(t, "echo $(basename foo/bar/bar_out) > $OUT", cmd.String(state, barTest))
}
