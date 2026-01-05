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

	bottom := core.NewBuildTarget(core.NewBuildLabel("pkg", "bottom"))
	bottom.AddLabel("target:bottom")
	bottom.SetState(core.Built)

	middle := core.NewBuildTarget(core.NewBuildLabel("pkg", "middle"))
	middle.AddDependency(bottom.Label)
	middle.AddLabel("target:middle")
	middle.SetState(core.Built)

	top := core.NewBuildTarget(core.NewBuildLabel("pkg", "top"))
	top.AddDependency(middle.Label)
	top.AddLabel("target:top")
	top.SetState(core.Built)

	state.Graph.AddTarget(bottom)
	state.Graph.AddTarget(middle)
	state.Graph.AddTarget(top)

	err := middle.ResolveDependencies(state.Graph)
	require.NoError(t, err)
	err = top.ResolveDependencies(state.Graph)
	require.NoError(t, err)

	s := &scope{state: state, pkg: core.NewPackage("pkg")}
	ls := getLabels(s, []pyObject{pyString(":top"), pyString("target:"), False, True, pyInt(-1)}).(pyList) // transitive=True
	assert.Equal(t, pyList{pyString("bottom"), pyString("middle"), pyString("top")}, ls)
	ls = getLabels(s, []pyObject{pyString(":top"), pyString("target:"), False, None, pyInt(-1)}).(pyList) // maxdepth=-1 (equivalent to above)
	assert.Equal(t, pyList{pyString("bottom"), pyString("middle"), pyString("top")}, ls)

	ls = getLabels(s, []pyObject{pyString(":top"), pyString("target:"), False, False, pyInt(-1)}).(pyList) // transitive=False
	assert.Equal(t, pyList{pyString("top")}, ls)
	ls = getLabels(s, []pyObject{pyString(":top"), pyString("target:"), False, None, pyInt(0)}).(pyList) // maxdepth=0 (equivalent to above)
	assert.Equal(t, pyList{pyString("top")}, ls)

	ls = getLabels(s, []pyObject{pyString(":top"), pyString("target:"), False, None, pyInt(1)}).(pyList) // maxdepth=1
	assert.Equal(t, pyList{pyString("middle"), pyString("top")}, ls)
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
