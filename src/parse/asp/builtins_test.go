package asp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thought-machine/please/src/core"
)

func TestPackageName(t *testing.T) {
	s := &scope{pkg: &core.Package{Name: "test/package"}}
	assert.Equal(t, "test/package", packageName(s, []pyObject{pyNone{}}).String())
	assert.Equal(t, "test/package", packageName(s, []pyObject{pyString(":test")}).String())
	assert.Equal(t, "foo/bar", packageName(s, []pyObject{pyString("//foo/bar:test")}).String())

	s = &scope{subincludeLabel: &core.BuildLabel{PackageName: "test/package"}}
	assert.Equal(t, "test/package", packageName(s, []pyObject{pyNone{}}).String())
	assert.Equal(t, "test/package", packageName(s, []pyObject{pyString(":test")}).String())
	assert.Equal(t, "foo/bar", packageName(s, []pyObject{pyString("//foo/bar:test")}).String())
}

func TestGetLabels(t *testing.T) {
	state := core.NewBuildState(core.DefaultConfiguration())
	foo := core.NewBuildTarget(core.NewBuildLabel("pkg", "foo"))
	foo.AddLabel("cc:ld:-ldl")
	foo.SetState(core.Built)

	bar := core.NewBuildTarget(core.NewBuildLabel("pkg", "bar"))
	bar.AddDependency(foo.Label)
	bar.AddLabel("cc:ld:-pthread")
	bar.SetState(core.Built)

	state.Graph.AddTarget(foo)
	state.Graph.AddTarget(bar)

	err := bar.ResolveDependencies(state.Graph)
	require.NoError(t, err)

	s := &scope{state: state, pkg: core.NewPackage("pkg")}
	ls := getLabels(s, []pyObject{pyString(":bar"), pyString("cc:ld:"), False, True}).(pyList)
	assert.Len(t, ls, 2)

	ls = getLabels(s, []pyObject{pyString(":bar"), pyString("cc:ld:"), False, False}).(pyList)
	assert.Len(t, ls, 1)
	assert.Equal(t, pyString("-pthread"), ls[0])
}
