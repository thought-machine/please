package query

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

func TestTargetsChanged(t *testing.T) {
	s1 := core.NewDefaultBuildState()
	s2 := core.NewDefaultBuildState()
	t1 := addTarget(s1, "//src/query:changes", nil, "src/query/changes.go")
	t2 := addTarget(s2, "//src/query:changes", nil, "src/query/changes.go")
	addTarget(s1, "//src/query:changes_test", t1, "src/query/changes_test.go")
	t4 := addTarget(s2, "//src/query:changes_test", t2, "src/query/changes_test.go")
	assert.EqualValues(t, []core.BuildLabel{}, DiffGraphs(s1, s2, nil))

	t2.Command = "nope nope nope"
	assert.EqualValues(t, []core.BuildLabel{t2.Label, t4.Label}, DiffGraphs(s1, s2, nil))

	t2.AddLabel("nope")
	s2.SetIncludeAndExclude(nil, []string{"nope"})
	assert.EqualValues(t, []core.BuildLabel{}, DiffGraphs(s1, s2, nil))
}

func addTarget(state *core.BuildState, label string, dep *core.BuildTarget, sources ...string) *core.BuildTarget {
	t := core.NewBuildTarget(core.ParseBuildLabel(label, ""))
	for _, src := range sources {
		t.AddSource(core.FileLabel{
			File:    src,
			Package: t.Label.PackageName,
		})
	}
	state.Graph.AddTarget(t)
	if dep != nil {
		t.AddDependency(dep.Label)
		state.Graph.AddDependency(t, dep.Label)
	}
	return t
}
