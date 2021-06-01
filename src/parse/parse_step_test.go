// Tests for general parse functions.

package parse

import (
	"github.com/stretchr/testify/assert"
	"github.com/thought-machine/please/src/core"
	"testing"
	"time"
)

const tid = 1

// TODO(jpoole): Use brain to figure out what we're actually waiting for here instead of just sleeping 100ms
func TestAddDepSimple(t *testing.T) {
	// Simple case with only one package parsed and one target added
	state := makeState(true, false)
	activateTarget(tid, state, nil, buildLabel("//package1:target1"), core.OriginalTarget, false)

	time.Sleep(time.Millisecond * 100)

	assertPendingParses(t, state, "//package2:target1", "//package2:target1")
	assertPendingBuilds(t, state) // None until package2 parses
	assert.Equal(t, 5, state.NumActive())
}

func TestAddDepMultiple(t *testing.T) {
	// Similar to above but doing all targets in that package
	state := makeState(true, false)
	activateTarget(tid, state, nil, buildLabel("//package1:target1"), core.OriginalTarget, false)
	activateTarget(tid, state, nil, buildLabel("//package1:target2"), core.OriginalTarget, false)
	activateTarget(tid, state, nil, buildLabel("//package1:target3"), core.OriginalTarget, false)

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
	activateTarget(tid, state, nil, buildLabel("//package1:target1"), core.OriginalTarget, false)

	time.Sleep(time.Millisecond * 100)

	assertPendingBuilds(t, state, "//package2:target2") // This is the only candidate target
	assertPendingParses(t, state)                       // None, we have both packages already
	assert.Equal(t, 6, state.NumActive())
}

func TestAddDepNoBuild(t *testing.T) {
	// Tag state as not needing build. We shouldn't get any pending builds at this point.
	state := makeState(true, true)
	state.NeedBuild = false
	activateTarget(tid, state, nil, buildLabel("//package1:target1"), core.OriginalTarget, false)

	time.Sleep(time.Millisecond * 100)

	assertPendingParses(t, state) // None, we have both packages already
	assertPendingBuilds(t, state) // Nothing because we don't need to build.
}

func TestAddParseDep(t *testing.T) {
	// Tag state as not needing build. Any target that needs to be built to complete parse
	// should still get queued for build though. Recall that we indicate this with :all...
	state := makeState(true, true)
	state.NeedBuild = false
	activateTarget(tid, state, nil, buildLabel("//package2:target2"), buildLabel("//package3:all"), false)

	time.Sleep(time.Millisecond * 100)

	assertPendingBuilds(t, state, "//package2:target2") // Queued because it's needed for parse
	assertPendingParses(t, state)                       // None, we have both packages already
	assert.Equal(t, 2, state.NumActive())
}

func TestAddDepRescan(t *testing.T) {
	t.Skip("Not convinced this test is a good reflection of reality")
	// Simulate a post-build function and rescan.
	state := makeState(true, true)
	activateTarget(tid, state, nil, buildLabel("//package1:target1"), core.OriginalTarget, false)

	time.Sleep(time.Millisecond * 100)

	assertPendingBuilds(t, state, "//package2:target2") // This is the only candidate target
	assertPendingParses(t, state)                       // None, we have both packages already
	assert.Equal(t, 6, state.NumActive())

	// Add new target & dep to target1
	target4 := makeTarget("//package1:target4")
	state.Graph.Package("package1", "").AddTarget(target4)
	state.Graph.AddTarget(target4)
	target1 := state.Graph.TargetOrDie(buildLabel("//package1:target1"))
	target1.AddDependency(buildLabel("//package1:target4"))

	// Fake test: calling this now should have no effect because rescan is not true.
	state.QueueTarget(buildLabel("//package1:target1"), core.OriginalTarget, false, false)
	assertPendingParses(t, state)
	assertPendingBuilds(t, state) // Note that the earlier call to assertPendingBuilds cleared it.

	// Now running this should activate it
	rescanDeps(state, map[*core.BuildTarget]struct{}{target1: {}})
	time.Sleep(time.Millisecond * 100)

	assertPendingBuilds(t, state, "//package1:target4")
	assertPendingParses(t, state)
}

func TestAddParseDepDeferred(t *testing.T) {
	t.Skip("Not convinced this test is a good reflection of reality")
	// Similar to TestAddParseDep but where we scan the package once and come back later because
	// something else asserts a dependency on it.
	state := makeState(true, true)
	state.NeedBuild = false
	assert.Equal(t, 1, state.NumActive())
	activateTarget(tid, state, nil, buildLabel("//package2:target2"), core.OriginalTarget, false)
	time.Sleep(time.Millisecond * 100)

	assertPendingParses(t, state)
	assertPendingBuilds(t, state) // Not yet.

	// Now the undefer kicks off...
	activateTarget(tid, state, nil, buildLabel("//package2:target2"), buildLabel("//package1:all"), false)
	time.Sleep(time.Millisecond * 100)

	assertPendingBuilds(t, state, "//package2:target2") // This time!
	assertPendingParses(t, state)
	assert.Equal(t, 2, state.NumActive())
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
	parses, _ := getAllPending(state)
	assert.ElementsMatch(t, targets, parses)
}

func assertPendingBuilds(t *testing.T, state *core.BuildState, targets ...string) {
	_, builds := getAllPending(state)
	assert.ElementsMatch(t, targets, builds)
}

func getAllPending(state *core.BuildState) ([]string, []string) {
	parses, builds, _, tests, _ := state.TaskQueues()
	state.Stop()
	var pendingParses, pendingBuilds []string
	for parses != nil || builds != nil || tests != nil {
		select {
		case p, ok := <-parses:
			if !ok {
				parses = nil
				break
			}
			pendingParses = append(pendingParses, p.Label.String())
		case l, ok := <-builds:
			if !ok {
				builds = nil
				break
			}
			pendingBuilds = append(pendingBuilds, l.String())
		case _, ok := <-tests:
			if !ok {
				tests = nil
				break
			}
		}
	}
	return pendingParses, pendingBuilds
}

func buildLabel(bl string) core.BuildLabel {
	return core.ParseBuildLabel(bl, "")
}
