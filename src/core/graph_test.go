package core

import (
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

// makeTarget3 creates a new build target for us.
func makeTarget3(label string, deps ...*BuildTarget) *BuildTarget {
	target := NewBuildTarget(ParseBuildLabel(label, ""))
	for _, dep := range deps {
		target.AddDependency(dep.Label)
	}
	return target
}
