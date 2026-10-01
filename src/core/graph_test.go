package core

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddTarget(t *testing.T) {
	graph := NewGraph()
	target := makeTarget3("//src/core:target1")
	graph.AddTarget(target)
	assert.Equal(t, target, graph.TargetOrDie(target.Label))
}

func TestAddPackage(t *testing.T) {
	graph := NewGraph()
	pkg := NewPackage("src/core")
	graph.AddPackage(pkg)
	assert.Equal(t, pkg, graph.Package("src/core", ""))
}

func TestTarget(t *testing.T) {
	graph := NewGraph()
	target := graph.Target(ParseBuildLabel("//src/core:target1", ""))
	assert.Nil(t, target)
	assert.Equal(t, 0, len(graph.AllTargets()))
}

func TestDependentTargets(t *testing.T) {
	graph := NewGraph()
	target1 := makeTarget3("//src/core:target1")
	target2 := makeTarget3("//src/core:target2")
	target3 := makeTarget3("//src/core:target3")
	target2.AddDependency(target1.Label)
	target1.AddDependency(target3.Label)
	target1.AddProvide("go", []BuildLabel{ParseBuildLabel(":target3", "src/core")})
	target2.Requires = append(target2.Requires, "go")
	graph.AddTarget(target1)
	graph.AddTarget(target2)
	graph.AddTarget(target3)
	assert.Equal(t, []BuildLabel{target3.Label}, graph.DependentTargets(target2.Label, target1.Label))
}

func TestSubrepo(t *testing.T) {
	graph := NewGraph()
	graph.AddSubrepo(&Subrepo{Name: "test", Root: "plz-out/gen/test"})
	subrepo := graph.Subrepo("test")
	assert.NotNil(t, subrepo)
	assert.Equal(t, "plz-out/gen/test", subrepo.Root)
}

func TestSubincludes(t *testing.T) {
	graph := NewGraph()
	pkg := ParseBuildLabel("//src/core:all", "")
	outer := ParseBuildLabel("//build_defs:outer", "")
	inner := ParseBuildLabel("//build_defs:inner", "")
	graph.AddSubinclude(pkg, outer)
	graph.AddSubinclude(outer, inner)

	// Subincludes is only the direct ones...
	assert.Equal(t, []BuildLabel{outer}, slices.Collect(graph.Subincludes(pkg)))
	assert.Equal(t, []BuildLabel{inner}, slices.Collect(graph.Subincludes(outer)))
	assert.Empty(t, slices.Collect(graph.Subincludes(inner)))
	// ...whereas AllSubincludes follows them transitively.
	assert.ElementsMatch(t, []BuildLabel{outer, inner}, slices.Collect(graph.AllSubincludes(pkg)))
	assert.Equal(t, []BuildLabel{inner}, slices.Collect(graph.AllSubincludes(outer)))
	assert.Empty(t, slices.Collect(graph.AllSubincludes(inner)))
}

func TestAddSubincludeIsIdempotent(t *testing.T) {
	graph := NewGraph()
	pkg := ParseBuildLabel("//src/core:all", "")
	defs := ParseBuildLabel("//build_defs:defs", "")
	// The same file is interpreted once per repo that includes it, so this happens routinely.
	graph.AddSubinclude(pkg, defs)
	graph.AddSubinclude(pkg, defs)
	assert.Equal(t, []BuildLabel{defs}, slices.Collect(graph.AllSubincludes(pkg)))
}

func TestAllSubincludesDiamond(t *testing.T) {
	graph := NewGraph()
	pkg := ParseBuildLabel("//src/core:all", "")
	left := ParseBuildLabel("//build_defs:left", "")
	right := ParseBuildLabel("//build_defs:right", "")
	common := ParseBuildLabel("//build_defs:common", "")
	graph.AddSubinclude(pkg, left)
	graph.AddSubinclude(pkg, right)
	graph.AddSubinclude(left, common)
	graph.AddSubinclude(right, common)

	// ElementsMatch compares as a multiset, so this fails if common is yielded twice.
	assert.ElementsMatch(t, []BuildLabel{left, right, common}, slices.Collect(graph.AllSubincludes(pkg)))
}

func TestAllSubincludesCycle(t *testing.T) {
	graph := NewGraph()
	a := ParseBuildLabel("//build_defs:a", "")
	b := ParseBuildLabel("//build_defs:b", "")
	graph.AddSubinclude(a, b)
	graph.AddSubinclude(b, a)

	// A cycle here would deadlock the parser, but the graph can be walked while it's still being
	// built (e.g. by cycle detection) so this mustn't recurse forever.
	assert.Equal(t, []BuildLabel{b}, slices.Collect(graph.AllSubincludes(a)))
}

// makeTarget3 creates a new build target for us.
func makeTarget3(label string, deps ...*BuildTarget) *BuildTarget {
	target := NewBuildTarget(ParseBuildLabel(label, ""))
	for _, dep := range deps {
		target.AddDependency(dep.Label)
	}
	return target
}
