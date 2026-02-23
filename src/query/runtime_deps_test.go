package query

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

func TestRuntimeDeps(t *testing.T) {
	var addTarget func(string, bool, bool, []*core.BuildTarget, []*core.BuildTarget, []*core.BuildTarget) *core.BuildTarget

	for _, test := range []struct {
		Description string
		GraphFunc   func() *core.BuildTarget
		Expected    []string
	}{
		{
			Description: "Explicit run-time deps",
			GraphFunc: func() *core.BuildTarget {
				run1 := addTarget("runtime_dep_1", false, false, nil, nil, nil)
				run2 := addTarget("runtime_dep_2", false, false, nil, nil, []*core.BuildTarget{run1})
				root := addTarget("root", false, false, nil, nil, []*core.BuildTarget{run2})
				return root
			},
			Expected: []string{
				"//test:runtime_dep_1",
				"//test:runtime_dep_2",
			},
		},
		{
			Description: "Run-time deps from srcs but not deps",
			GraphFunc: func() *core.BuildTarget {
				run1 := addTarget("runtime_dep_1", false, false, nil, nil, nil)
				run2 := addTarget("runtime_dep_2", false, false, nil, nil, []*core.BuildTarget{run1})
				src := addTarget("src", false, false, nil, nil, []*core.BuildTarget{run2})
				run3 := addTarget("runtime_dep_3", false, false, nil, nil, nil)
				run4 := addTarget("runtime_dep_4", false, false, nil, nil, []*core.BuildTarget{run3})
				dep := addTarget("dep", false, false, nil, nil, []*core.BuildTarget{run4})
				root := addTarget("root", true, false, []*core.BuildTarget{src}, []*core.BuildTarget{dep}, nil)
				return root
			},
			Expected: []string{
				"//test:runtime_dep_1",
				"//test:runtime_dep_2",
			},
		},
		{
			Description: "Run-time deps from deps but not srcs",
			GraphFunc: func() *core.BuildTarget {
				run1 := addTarget("runtime_dep_1", false, false, nil, nil, nil)
				run2 := addTarget("runtime_dep_2", false, false, nil, nil, []*core.BuildTarget{run1})
				src := addTarget("src", false, false, nil, nil, []*core.BuildTarget{run2})
				run3 := addTarget("runtime_dep_3", false, false, nil, nil, nil)
				run4 := addTarget("runtime_dep_4", false, false, nil, nil, []*core.BuildTarget{run3})
				dep := addTarget("dep", false, false, nil, nil, []*core.BuildTarget{run4})
				root := addTarget("root", false, true, []*core.BuildTarget{src}, []*core.BuildTarget{dep}, nil)
				return root
			},
			Expected: []string{
				"//test:runtime_dep_3",
				"//test:runtime_dep_4",
			},
		},
		{
			Description: "Run-time deps from both srcs and deps",
			GraphFunc: func() *core.BuildTarget {
				run1 := addTarget("runtime_dep_1", false, false, nil, nil, nil)
				run2 := addTarget("runtime_dep_2", false, false, nil, nil, []*core.BuildTarget{run1})
				src := addTarget("src", false, false, nil, nil, []*core.BuildTarget{run2})
				run3 := addTarget("runtime_dep_3", false, false, nil, nil, nil)
				run4 := addTarget("runtime_dep_4", false, false, nil, nil, []*core.BuildTarget{run3})
				dep := addTarget("dep", false, false, nil, nil, []*core.BuildTarget{run4})
				root := addTarget("root", true, true, []*core.BuildTarget{src}, []*core.BuildTarget{dep}, nil)
				return root
			},
			Expected: []string{
				"//test:runtime_dep_1",
				"//test:runtime_dep_2",
				"//test:runtime_dep_3",
				"//test:runtime_dep_4",
			},
		},
	} {
		t.Run(test.Description, func(t *testing.T) {
			state := core.NewDefaultBuildState()
			testPkg := core.NewPackage("test")

			addTarget = func(name string, fromSrcs, fromDeps bool, srcs, deps, runtimeDeps []*core.BuildTarget) *core.BuildTarget {
				t := addNewTarget(state.Graph, testPkg, name, nil)
				t.RuntimeDependenciesFromSources = fromSrcs
				t.RuntimeDependenciesFromDependencies = fromDeps
				for _, src := range srcs {
					t.AddSource(src.Label)
				}
				for _, dep := range deps {
					t.AddDependency(dep.Label)
				}
				if runtimeDeps != nil {
					t.IsBinary = true
					for _, runtimeDep := range runtimeDeps {
						t.AddMaybeExportedDependency(runtimeDep.Label, false, false, false, true)
					}
				}
				return t
			}

			root := test.GraphFunc()

			var buf bytes.Buffer
			RuntimeDeps(&buf, state, []core.BuildLabel{root.Label})
			assert.ElementsMatch(t, test.Expected, strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n"))
		})
	}
}
