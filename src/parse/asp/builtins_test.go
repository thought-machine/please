package asp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thought-machine/please/src/core"
)

func TestPackageName(t *testing.T) {
	s := &scope{
		pkg:   &core.Package{Name: "test/package"},
		state: core.NewBuildState(core.DefaultConfiguration()),
	}
	assert.Equal(t, "test/package", packageName(s, []pyObject{pyNone{}, pyNone{}}).String())
	assert.Equal(t, "test/package", packageName(s, []pyObject{pyString(":test"), pyNone{}}).String())
	assert.Equal(t, "foo/bar", packageName(s, []pyObject{pyString("//foo/bar:test"), pyNone{}}).String())
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

func TestTag(t *testing.T) {
	res := tag(nil, []pyObject{pyString("name"), pyString("foo")})
	assert.Equal(t, res.String(), "_name#foo")

	res = tag(nil, []pyObject{res, pyString("bar")})
	assert.Equal(t, res.String(), "_name#foo_bar")
}

func TestObjLen(t *testing.T) {
	l := pyList{pyInt(1)}
	assert.EqualValues(t, 1, objLen(l))
	assert.EqualValues(t, 1, objLen(l.Freeze()))
	d := pyDict{"1": pyInt(1)}
	assert.EqualValues(t, 1, objLen(d))
	assert.EqualValues(t, 1, objLen(d.Freeze()))
}
