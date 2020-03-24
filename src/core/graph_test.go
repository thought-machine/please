package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddTarget(t *testing.T) {
	graph := NewGraph()
	target := makeTarget("//src/core:target1")
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

func TestRevDeps(t *testing.T) {
	graph := NewGraph()
	target1 := makeTarget("//src/core:target1")
	target2 := makeTarget("//src/core:target2", target1)
	target3 := makeTarget("//src/core:target3", target2)
	graph.AddTarget(target1)
	graph.AddTarget(target2)
	graph.AddTarget(target3)
	graph.AddDependency(target2, target1.Label)
	graph.AddDependency(target3, target2.Label)
	assert.Equal(t, []*BuildTarget{target2}, graph.ReverseDependencies(target1))
	assert.Equal(t, []*BuildTarget{target3}, graph.ReverseDependencies(target2))
	assert.Equal(t, 0, len(graph.ReverseDependencies(target3)))
}

func TestAllDepsBuilt(t *testing.T) {
	graph := NewGraph()
	target1 := makeTarget("//src/core:target1")
	target2 := makeTarget("//src/core:target2", target1)
	graph.AddTarget(target1)
	graph.AddTarget(target2)
	graph.AddDependency(target2, target1.Label)
	assert.True(t, target1.AllDepsBuilt(), "Should be true because it has no dependencies")
	assert.False(t, target2.AllDepsBuilt(), "Should be false because target1 isn't built yet")
	target1.SyncUpdateState(Inactive, Building)
	assert.False(t, target2.AllDepsBuilt(), "Should be false because target1 is building now")
	target1.SyncUpdateState(Building, Built)
	assert.True(t, target2.AllDepsBuilt(), "Should be true now target1 is built.")
}

func TestAllDepsResolved(t *testing.T) {
	graph := NewGraph()
	target1 := makeTarget("//src/core:target1")
	target2 := makeTarget("//src/core:target2")
	target2.AddDependency(target1.Label)
	graph.AddTarget(target2)
	assert.False(t, target2.AllDependenciesResolved(), "Haven't added a proper dep for target2 yet.")
	graph.AddTarget(target1)
	graph.AddDependency(target2, target1.Label)
	assert.True(t, target1.AllDependenciesResolved(), "Has no dependencies so they're all resolved")
	assert.True(t, target2.AllDependenciesResolved(), "Should be resolved now we've added target1.")
}

func TestDependentTargets(t *testing.T) {
	graph := NewGraph()
	target1 := makeTarget("//src/core:target1")
	target2 := makeTarget("//src/core:target2")
	target3 := makeTarget("//src/core:target3")
	target2.AddDependency(target1.Label)
	target1.AddDependency(target3.Label)
	target1.AddProvide("go", ParseBuildLabel(":target3", "src/core"))
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

// makeTarget creates a new build target for us.
func makeTarget(label string, deps ...*BuildTarget) *BuildTarget {
	target := NewBuildTarget(ParseBuildLabel(label, ""))
	for _, dep := range deps {
		target.AddDependency(dep.Label)
	}
	return target
}

func TestFindGraphCycle(t *testing.T) {
	graph := NewGraph()
	graph.AddTarget(makeTarget2("//src/output:target1", "//src/output:target2", "//src/output:target3", "//src/core:target2"))
	graph.AddTarget(makeTarget2("//src/output:target2", "//src/output:target3", "//src/core:target1"))
	graph.AddTarget(makeTarget2("//src/output:target3"))
	graph.AddTarget(makeTarget2("//src/core:target1", "//third_party/go:target2", "//third_party/go:target3", "//src/core:target3"))
	graph.AddTarget(makeTarget2("//src/core:target2", "//third_party/go:target3", "//src/output:target2"))
	graph.AddTarget(makeTarget2("//src/core:target3", "//third_party/go:target2", "//src/output:target2"))
	graph.AddTarget(makeTarget2("//third_party/go:target2", "//third_party/go:target1"))
	graph.AddTarget(makeTarget2("//third_party/go:target3", "//third_party/go:target1"))
	graph.AddTarget(makeTarget2("//third_party/go:target1"))
	for _, target := range graph.AllTargets() {
		for _, dep := range target.DeclaredDependencies() {
			graph.AddDependency(target, dep)
		}
	}
	cycle := graph.detectCycle(map[BuildLabel]struct{}{}, graph.TargetOrDie(ParseBuildLabel("//src/output:target1", "")), nil)
	if len(cycle) == 0 {
		t.Fatalf("Failed to find cycle")
	} else if len(cycle) != 3 {
		t.Errorf("Found unexpected cycle of length %d", len(cycle))
	}
	assertTarget(t, cycle[0], "//src/output:target2")
	assertTarget(t, cycle[1], "//src/core:target1")
	assertTarget(t, cycle[2], "//src/core:target3")
}

// Factory function for build targets
func makeTarget2(label string, deps ...string) *BuildTarget {
	target := NewBuildTarget(ParseBuildLabel(label, ""))
	for _, dep := range deps {
		target.AddDependency(ParseBuildLabel(dep, ""))
	}
	return target
}

func assertTarget(t *testing.T, target *BuildTarget, label string) {
	if target.Label != ParseBuildLabel(label, "") {
		t.Errorf("Unexpected target in detected cycle; expected %s, was %s", label, target.Label)
	}
}
