package query

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

func TestQueryDeps(t *testing.T) {
	state := core.NewDefaultBuildState()
	pkg1 := core.NewPackage("tools/performance")
	pkg2 := core.NewPackage("third_party/python")

	addTarget := func(pkg *core.Package, name string, deps ...*core.BuildTarget) *core.BuildTarget {
		t := addNewTarget(state.Graph, pkg, name, nil)
		for _, dep := range deps {
			t.AddDependency(dep.Label)
		}
		return t
	}

	t1 := addTarget(pkg2, "_six#download")
	t2 := addTarget(pkg2, "_six#wheel", t1)
	t3 := addTarget(pkg2, "six", t2)
	t4 := addTarget(pkg2, "_absl#download")
	t5 := addTarget(pkg2, "_absl#wheel", t4, t3)
	t6 := addTarget(pkg2, "absl", t5)
	t7 := addTarget(pkg2, "_colorlog#download")
	t8 := addTarget(pkg2, "_colorlog#wheel", t7)
	t9 := addTarget(pkg2, "colorlog", t8)

	t10 := addTarget(pkg1, "_parse_perf_test#pex")
	t11 := addTarget(pkg1, "_parse_perf_test#lib_zip")
	t12 := addTarget(pkg1, "_parse_perf_test#lib", t11)
	t13 := addTarget(pkg1, "parse_perf_test", t10, t12, t6, t9)
	query := []core.BuildLabel{t13.Label}

	t.Run("visible_level_1", func(t *testing.T) {
		var buf bytes.Buffer
		Deps(&buf, state, query, false, 1, false)
		assert.Equal(t, `//third_party/python:absl
//third_party/python:colorlog
`, buf.String())
	})

	t.Run("visible_level_2", func(t *testing.T) {
		var buf bytes.Buffer
		Deps(&buf, state, query, false, 2, false)
		assert.Equal(t, `//third_party/python:absl
  //third_party/python:six
//third_party/python:colorlog
`, buf.String())
	})

	t.Run("visible_minus_level", func(t *testing.T) {
		var buf bytes.Buffer
		Deps(&buf, state, query, false, -1, false)
		assert.Equal(t, `//third_party/python:absl
  //third_party/python:six
//third_party/python:colorlog
`, buf.String())
	})

	t.Run("hidden_level_1", func(t *testing.T) {
		var buf bytes.Buffer
		Deps(&buf, state, query, true, 1, false)
		assert.Equal(t, `//third_party/python:absl
//third_party/python:colorlog
//tools/performance:_parse_perf_test#lib
//tools/performance:_parse_perf_test#pex
`, buf.String())
	})

	t.Run("hidden_level_2", func(t *testing.T) {
		var buf bytes.Buffer
		Deps(&buf, state, query, true, 2, false)
		assert.Equal(t, `//third_party/python:absl
  //third_party/python:_absl#wheel
//third_party/python:colorlog
  //third_party/python:_colorlog#wheel
//tools/performance:_parse_perf_test#lib
  //tools/performance:_parse_perf_test#lib_zip
//tools/performance:_parse_perf_test#pex
`, buf.String())
	})

	t.Run("hidden_minus_level", func(t *testing.T) {
		var buf bytes.Buffer
		Deps(&buf, state, query, true, -1, false)
		assert.Equal(t, `//third_party/python:absl
  //third_party/python:_absl#wheel
    //third_party/python:_absl#download
    //third_party/python:six
      //third_party/python:_six#wheel
        //third_party/python:_six#download
//third_party/python:colorlog
  //third_party/python:_colorlog#wheel
    //third_party/python:_colorlog#download
//tools/performance:_parse_perf_test#lib
  //tools/performance:_parse_perf_test#lib_zip
//tools/performance:_parse_perf_test#pex
`, buf.String())
	})
}
