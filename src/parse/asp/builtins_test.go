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

func TestStrFormat(t *testing.T) {
	s := &scope{
		locals: map[string]pyObject{
			"spam": pyString("abc"),
			"eggs": pyString("def"),
		},
	}

	assert.EqualValues(t, "test 123 abc ${wibble} 456 def {wobble}", strFormat(s, []pyObject{
		pyString("test {} {spam} ${wibble} {} {eggs} {wobble}"), pyString("123"), pyString("456"),
	}))
}

func TestStrFormat2(t *testing.T) {
	s := &scope{
		locals: map[string]pyObject{
			"owner":    pyString("please-build"),
			"plugin":   pyString("java-rules"),
			"revision": pyString("v0.3.0"),
		},
	}

	assert.EqualValues(t, "https://github.com/please-build/java-rules/archive/v0.3.0.zip", strFormat(s, []pyObject{
		pyString("https://github.com/{owner}/{plugin}/archive/{revision}.zip"),
	}))
}

func TestStrFormat3(t *testing.T) {
	s := &scope{
		locals: map[string]pyObject{
			"url_base":     pyString("https://please.build/py-wheels"),
			"package_name": pyString("coverage"),
			"version":      pyString("5.5"),
		},
	}

	assert.EqualValues(t, "https://please.build/py-wheels/coverage-5.5-${OS}_${ARCH}.whl", strFormat(s, []pyObject{
		pyString("{url_base}/{package_name}-{version}-${{OS}}_${{ARCH}}.whl"),
	}))
}
