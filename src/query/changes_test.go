package query

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

func TestDiffGraphs(t *testing.T) {
	s1 := core.NewDefaultBuildState()
	s2 := core.NewDefaultBuildState()
	t1 := addTarget(s1, "//src/query:changes", nil, "src/query/changes.go")
	t2 := addTarget(s2, "//src/query:changes", nil, "src/query/changes.go")
	addTarget(s1, "//src/query:changes_test", t1, "src/query/changes_test.go")
	t4 := addTarget(s2, "//src/query:changes_test", t2, "src/query/changes_test.go")
	assert.EqualValues(t, []core.BuildLabel{}, DiffGraphs(s1, s2, nil, true, true))

	t2.Command = "nope nope nope"
	assert.EqualValues(t, []core.BuildLabel{t2.Label, t4.Label}, DiffGraphs(s1, s2, nil, true, true))

	t2.AddLabel("nope")
	t4.AddLabel("test")
	s2.SetIncludeAndExclude(nil, []string{"nope", "test"})
	assert.EqualValues(t, []core.BuildLabel{}, DiffGraphs(s1, s2, nil, true, true))
}

func TestDiffGraphsIncludeNothing(t *testing.T) {
	s1 := core.NewDefaultBuildState()
	s2 := core.NewDefaultBuildState()
	t1 := addTarget(s1, "//src/core:core", nil, "src/core/core.go")
	t2 := addTarget(s1, "//src/query:changes", t1, "src/query/changes.go")
	addTarget(s1, "//src/query:changes_test", t2, "src/query/changes_test.go")
	t1 = addTarget(s2, "//src/core:core", nil, "src/core/core_changed.go")
	t2 = addTarget(s2, "//src/query:changes", t1, "src/query/changes.go")
	addTarget(s2, "//src/query:changes_test", t2, "src/query/changes_test.go")
	assert.EqualValues(t, []core.BuildLabel{t1.Label}, DiffGraphs(s1, s2, nil, false, false))
}

func TestDiffGraphsIncludeDirect(t *testing.T) {
	s1 := core.NewDefaultBuildState()
	s2 := core.NewDefaultBuildState()
	t1 := addTarget(s1, "//src/core:core", nil, "src/core/core.go")
	t2 := addTarget(s1, "//src/query:changes", t1, "src/query/changes.go")
	addTarget(s1, "//src/query:changes_test", t2, "src/query/changes_test.go")
	t1 = addTarget(s2, "//src/core:core", nil, "src/core/core_changed.go")
	t2 = addTarget(s2, "//src/query:changes", t1, "src/query/changes.go")
	addTarget(s2, "//src/query:changes_test", t2, "src/query/changes_test.go")
	assert.EqualValues(t, []core.BuildLabel{t1.Label, t2.Label}, DiffGraphs(s1, s2, nil, true, false))
}

func TestDiffGraphsIncludeTransitive(t *testing.T) {
	s1 := core.NewDefaultBuildState()
	s2 := core.NewDefaultBuildState()
	t1 := addTarget(s1, "//src/core:core", nil, "src/core/core.go")
	t2 := addTarget(s1, "//src/query:changes", t1, "src/query/changes.go")
	t3 := addTarget(s1, "//src/query:changes_test", t2, "src/query/changes_test.go")
	t1 = addTarget(s2, "//src/core:core", nil, "src/core/core_changed.go")
	t2 = addTarget(s2, "//src/query:changes", t1, "src/query/changes.go")
	t3 = addTarget(s2, "//src/query:changes_test", t2, "src/query/changes_test.go")
	assert.EqualValues(t, []core.BuildLabel{t1.Label, t2.Label, t3.Label}, DiffGraphs(s1, s2, nil, true, true))
}

func TestChangesIncludesDataDirs(t *testing.T) {
	s := core.NewDefaultBuildState()
	t1 := addTarget(s, "//src/core:core", nil, "src/core/core.go")
	t2 := addTarget(s, "//src/query:changes", t1, "src/query/changes.go")
	t3 := addTarget(s, "//src/query:changes_test", t2, "src/query/changes_test.go")
	t3.AddDatum(core.FileLabel{Package: "src/query", File: "test_data"})
	assert.EqualValues(t, []core.BuildLabel{t3.Label}, Changes(s, []string{"src/query/test_data/some_dir/test_file1.txt"}, false, false))
}

func addTarget(state *core.BuildState, label string, dep *core.BuildTarget, sources ...string) *core.BuildTarget {
	t := core.NewBuildTarget(core.ParseBuildLabel(label, ""))
	for _, src := range sources {
		t.AddSource(core.FileLabel{
			File:    strings.TrimLeft(strings.TrimPrefix(src, t.Label.PackageName), "/"),
			Package: t.Label.PackageName,
		})
	}
	state.Graph.AddTarget(t)
	if dep != nil {
		t.AddDependency(dep.Label)
		state.Graph.AddDependency(t.Label, dep.Label)
	}
	pkg := state.Graph.PackageByLabel(t.Label)
	if pkg == nil {
		pkg = core.NewPackage(t.Label.PackageName)
		state.Graph.AddPackage(pkg)
	}
	pkg.AddTarget(t)
	return t
}
