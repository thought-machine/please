package query

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
)

func TestDiffGraphs(t *testing.T) {
	s1 := core.NewDefaultBuildState()
	s2 := core.NewDefaultBuildState()
	t1 := addTarget(s1, "//src/query:changes", nil, "src/query/changes.go")
	t2 := addTarget(s2, "//src/query:changes", nil, "src/query/changes.go")
	addTarget(s1, "//src/query:changes_test", t1, "src/query/changes_test.go")
	t4 := addTarget(s2, "//src/query:changes_test", t2, "src/query/changes_test.go")
	assert.EqualValues(t, []core.BuildLabel{}, DiffGraphs(s1, s2, nil, -1, false))

	t2.Command = "nope nope nope"
	assert.EqualValues(t, []core.BuildLabel{t2.Label, t4.Label}, DiffGraphs(s1, s2, nil, -1, false))

	t2.AddLabel("nope")
	t4.AddLabel("test")
	s2.SetIncludeAndExclude(nil, []string{"nope", "test"})
	assert.EqualValues(t, []core.BuildLabel{}, DiffGraphs(s1, s2, nil, -1, false))
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
	assert.EqualValues(t, []core.BuildLabel{t1.Label}, DiffGraphs(s1, s2, nil, 0, false))
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
	assert.EqualValues(t, []core.BuildLabel{t1.Label, t2.Label}, DiffGraphs(s1, s2, nil, 1, false))
}

func TestDiffGraphsLevel(t *testing.T) {
	s1 := core.NewDefaultBuildState()
	s2 := core.NewDefaultBuildState()
	t1 := addTarget(s1, "//src/core:core", nil, "src/core/core.go")
	t2 := addTarget(s1, "//src/query:changes", t1, "src/query/changes.go")
	t3 := addTarget(s1, "//src/query:changes_test", t2, "src/query/changes_test.go")
	addTarget(s1, "//src/query:changes_test2", t3, "src/query/changes_test2.go")
	t1 = addTarget(s2, "//src/core:core", nil, "src/core/core_changed.go")
	t2 = addTarget(s2, "//src/query:changes", t1, "src/query/changes.go")
	t3 = addTarget(s2, "//src/query:changes_test", t2, "src/query/changes_test.go")
	addTarget(s2, "//src/query:changes_test2", t3, "src/query/changes_test2.go")
	assert.EqualValues(t, []core.BuildLabel{t1.Label, t2.Label, t3.Label}, DiffGraphs(s1, s2, nil, 2, false))
}

func TestDiffGraphsIncludeTransitive(t *testing.T) {
	s1 := core.NewDefaultBuildState()
	s2 := core.NewDefaultBuildState()
	t1 := addTarget(s1, "//src/core:core", nil, "src/core/core.go")
	t2 := addTarget(s1, "//src/query:changes", t1, "src/query/changes.go")
	addTarget(s1, "//src/query:changes_test", t2, "src/query/changes_test.go")
	t1 = addTarget(s2, "//src/core:core", nil, "src/core/core_changed.go")
	t2 = addTarget(s2, "//src/query:changes", t1, "src/query/changes.go")
	t3 := addTarget(s2, "//src/query:changes_test", t2, "src/query/changes_test.go")
	assert.EqualValues(t, core.BuildLabels{t1.Label, t2.Label, t3.Label}, DiffGraphs(s1, s2, nil, -1, false))
}

func TestDiffGraphsStopsAtSubrepos(t *testing.T) {
	s1 := core.NewDefaultBuildState()
	t1 := addTarget(s1, "//:modfile", nil, "go.mod")
	t2 := addTarget(s1, "//third_party/go:mod", t1)
	t3 := addTarget(s1, "///third_party/go/mod//:mod", nil)
	t3.Subrepo = core.NewSubrepo(s1, "go_mod", "third_party/go", t2, cli.Arch{}, false)
	addTarget(s1, "//src/core:core", t3)

	s2 := core.NewDefaultBuildState()
	t1 = addTarget(s2, "//:modfile", nil, "go.mod")
	t2 = addTarget(s2, "//third_party/go:mod", t1)
	t3 = addTarget(s2, "///third_party/go/mod//:mod", nil)
	t3.Subrepo = core.NewSubrepo(s2, "go_mod", "third_party/go", t2, cli.Arch{}, false)
	addTarget(s2, "//src/core:core", t3)

	// t3 should not be changed here - its subrepo has but we should see that the targets generated in it are still identical
	assert.EqualValues(t, []core.BuildLabel{t1.Label, t2.Label}, DiffGraphs(s1, s2, []string{"go.mod"}, -1, false))
}

func TestDiffGraphsStillChecksTargetsInSubrepos(t *testing.T) {
	s1 := core.NewDefaultBuildState()
	t1 := addTarget(s1, "//:modfile", nil, "go.mod")
	t2 := addTarget(s1, "//third_party/go:mod", t1)
	t3 := addTarget(s1, "///third_party/go/mod//:mod", nil)
	t3.Subrepo = core.NewSubrepo(s1, "go_mod", "third_party/go", t2, cli.Arch{}, false)
	addTarget(s1, "//src/core:core", t3)

	s2 := core.NewDefaultBuildState()
	t1 = addTarget(s2, "//:modfile", nil, "go.mod")
	t2 = addTarget(s2, "//third_party/go:mod", t1)
	t3 = addTarget(s2, "///third_party/go/mod//:mod", nil, "test.go")
	t3.Subrepo = core.NewSubrepo(s2, "go_mod", "third_party/go", t2, cli.Arch{}, false)
	t4 := addTarget(s2, "//src/core:core", t3)

	// t3 should now count as changed - it has a different source file - and that should propagate to t4
	assert.EqualValues(t, []core.BuildLabel{t1.Label, t4.Label, t2.Label, t3.Label}, DiffGraphs(s1, s2, []string{"go.mod"}, -1, true))
	// If includeSubrepos=false, t4 should still count as changed, although we won't see t3.
	assert.EqualValues(t, []core.BuildLabel{t1.Label, t4.Label, t2.Label}, DiffGraphs(s1, s2, []string{"go.mod"}, -1, false))
}

func TestChangesIncludesDataDirs(t *testing.T) {
	s := core.NewDefaultBuildState()
	t1 := addTarget(s, "//src/core:core", nil, "src/core/core.go")
	t2 := addTarget(s, "//src/query:changes", t1, "src/query/changes.go")
	t3 := addTarget(s, "//src/query:changes_test", t2, "src/query/changes_test.go")
	t3.AddDatum(core.FileLabel{Package: "src/query", File: "test_data"})
	assert.EqualValues(t, []core.BuildLabel{t3.Label}, Changes(s, []string{"src/query/test_data/some_dir/test_file1.txt"}, 0, false))
}

func TestSameToolHashNoChange(t *testing.T) {
	s1 := core.NewDefaultBuildState()
	s2 := core.NewDefaultBuildState()
	target := addTarget(s1, "//src/core:core", nil, "src/core/core.go")
	target.AddTool(core.SystemPathLabel{Name: "non-existent", Path: s1.Config.Path()})
	target = addTarget(s2, "//src/core:core", nil, "src/core/core.go")
	target.AddTool(core.SystemPathLabel{Name: "non-existent", Path: s2.Config.Path()})
	assert.EqualValues(t, []core.BuildLabel{}, DiffGraphs(s1, s2, nil, -1, false))
}

func TestChangesIncludesRootTarget(t *testing.T) {
	s := core.NewDefaultBuildState()
	t1 := addTarget(s, "//:file", nil, "file.go")
	assert.EqualValues(t, []core.BuildLabel{t1.Label}, Changes(s, []string{"file.go"}, 0, false))
}

func addTarget(state *core.BuildState, label string, dep *core.BuildTarget, sources ...string) *core.BuildTarget {
	t := core.NewBuildTarget(core.ParseBuildLabel(label, ""))
	for _, src := range sources {
		t.AddSource(core.FileLabel{
			File:    strings.TrimLeft(strings.TrimPrefix(src, t.Label.PackageName), "/"),
			Package: t.Label.PackageName,
		})
	}
	if dep != nil {
		t.AddDependency(dep.Label)
	}
	state.Graph.AddTarget(t)
	if err := t.ResolveDependencies(state.Graph); err != nil {
		log.Fatalf("Failed to resolve dependency %s -> %s: %s", t, dep, err)
	}
	pkg := state.Graph.PackageByLabel(t.Label)
	if pkg == nil {
		pkg = core.NewPackageSubrepo(t.Label.PackageName, t.Label.Subrepo)
		state.Graph.AddPackage(pkg)
	}
	pkg.AddTarget(t)
	return t
}
