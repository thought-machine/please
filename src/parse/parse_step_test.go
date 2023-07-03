// Tests for general parse functions.

package parse

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

func TestAddDepSimple(t *testing.T) {
	// Simple case with only one package parsed and one target added
	state := makeState(true, false)
	state.ActivateTarget(nil, buildLabel("//package1:target1"), core.OriginalTarget, core.ParseModeNormal)

	time.Sleep(time.Millisecond * 100)

	assertPendingParses(t, state, "//package2:target1", "//package2:target1")
	assertPendingBuilds(t, state) // None until package2 parses
	assert.Equal(t, 5, state.NumActive())
}

func TestAddDepMultiple(t *testing.T) {
	// Similar to above but doing all targets in that package
	state := makeState(true, false)
	state.ActivateTarget(nil, buildLabel("//package1:target1"), core.OriginalTarget, core.ParseModeNormal)
	state.ActivateTarget(nil, buildLabel("//package1:target2"), core.OriginalTarget, core.ParseModeNormal)
	state.ActivateTarget(nil, buildLabel("//package1:target3"), core.OriginalTarget, core.ParseModeNormal)

	time.Sleep(time.Millisecond * 100)

	// We get an additional dep on target2, but not another on package2:target1 because target2
	// is already activated since package1:target1 depends on it
	assertPendingParses(t, state, "//package2:target1", "//package2:target1", "//package2:target2")
	assertPendingBuilds(t, state) // None until package2 parses
	assert.Equal(t, 7, state.NumActive())
}

func TestAddDepMultiplePackages(t *testing.T) {
	// This time we already have package2 parsed
	state := makeState(true, true)
	state.ActivateTarget(nil, buildLabel("//package1:target1"), core.OriginalTarget, core.ParseModeNormal)

	time.Sleep(time.Millisecond * 100)

	assertPendingBuilds(t, state, "//package2:target2") // This is the only candidate target
	assertPendingParses(t, state)                       // None, we have both packages already
	assert.Equal(t, 6, state.NumActive())
}

func TestAddDepNoBuild(t *testing.T) {
	// Tag state as not needing build. We shouldn't get any pending builds at this point.
	state := makeState(true, true)
	state.NeedBuild = false
	state.ActivateTarget(nil, buildLabel("//package1:target1"), core.OriginalTarget, core.ParseModeNormal)

	time.Sleep(time.Millisecond * 100)

	assertPendingParses(t, state) // None, we have both packages already
	assertPendingBuilds(t, state) // Nothing because we don't need to build.
}

func TestAddParseDep(t *testing.T) {
	// Tag state as not needing build. Any target that needs to be built to complete parse
	// should still get queued for build though. Recall that we indicate this with :all...
	state := makeState(true, true)
	state.NeedBuild = false
	state.ActivateTarget(nil, buildLabel("//package2:target2"), buildLabel("//package3:all"), core.ParseModeNormal)

	time.Sleep(time.Millisecond * 100)

	assertPendingBuilds(t, state, "//package2:target2") // Queued because it's needed for parse
	assertPendingParses(t, state)                       // None, we have both packages already
	assert.Equal(t, 2, state.NumActive())
}

func TestBuildFileNames(t *testing.T) {
	assert.Equal(t, "BUILD", buildFileNames([]string{"BUILD"}))
	assert.Equal(t, "BUILD or BUILD.plz", buildFileNames([]string{"BUILD", "BUILD.plz"}))
	assert.Equal(t, "BUILD, BUILD.plz or BUILD.test", buildFileNames([]string{"BUILD", "BUILD.plz", "BUILD.test"}))
}

func makeTarget(label string, deps ...string) *core.BuildTarget {
	target := core.NewBuildTarget(core.ParseBuildLabel(label, ""))
	for _, dep := range deps {
		target.AddDependency(core.ParseBuildLabel(dep, ""))
	}
	return target
}

// makeState creates a new build state with optionally one or two packages in it.
// Used in various tests above.
func makeState(withPackage1, withPackage2 bool) *core.BuildState {
	state := core.NewDefaultBuildState()
	if withPackage1 {
		pkg := core.NewPackage("package1")
		state.Graph.AddPackage(pkg)
		pkg.AddTarget(makeTarget("//package1:target1", "//package1:target2", "//package2:target1"))
		pkg.AddTarget(makeTarget("//package1:target2", "//package2:target1"))
		pkg.AddTarget(makeTarget("//package1:target3", "//package2:target2"))
		state.Graph.AddTarget(pkg.Target("target1"))
		state.Graph.AddTarget(pkg.Target("target2"))
		state.Graph.AddTarget(pkg.Target("target3"))
		addDeps(state.Graph, pkg)
	}
	if withPackage2 {
		pkg := core.NewPackage("package2")
		state.Graph.AddPackage(pkg)
		pkg.AddTarget(makeTarget("//package2:target1", "//package2:target2", "//package1:target3"))
		pkg.AddTarget(makeTarget("//package2:target2"))
		state.Graph.AddTarget(pkg.Target("target1"))
		state.Graph.AddTarget(pkg.Target("target2"))
		addDeps(state.Graph, pkg)
	}
	return state
}

func addDeps(graph *core.BuildGraph, pkg *core.Package) {
	for _, target := range pkg.AllTargets() {
		for _, dep := range target.DeclaredDependencies() {
			target.AddDependency(dep)
		}
	}
}

func assertPendingParses(t *testing.T, state *core.BuildState, targets ...string) {
	t.Helper()
	parses, _ := getAllPending(state)
	assert.ElementsMatch(t, targets, parses)
}

func assertPendingBuilds(t *testing.T, state *core.BuildState, targets ...string) {
	t.Helper()
	_, builds := getAllPending(state)
	assert.ElementsMatch(t, targets, builds)
}

func getAllPending(state *core.BuildState) ([]string, []string) {
	parses, builds := state.TaskQueues()
	state.Stop()
	var pendingParses, pendingBuilds []string
	for parses != nil || builds != nil {
		select {
		case p, ok := <-parses:
			if !ok {
				parses = nil
				break
			}
			pendingParses = append(pendingParses, p.Label.String())
		case t, ok := <-builds:
			if !ok {
				builds = nil
				break
			}
			pendingBuilds = append(pendingBuilds, t.Target.Label.String())
		}
	}
	return pendingParses, pendingBuilds
}

func buildLabel(bl string) core.BuildLabel {
	return core.ParseBuildLabel(bl, "")
}
