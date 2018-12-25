// Stress test around the build step stuff, specifically trying to
// identify concurrent map read / writes.

package build

import (
	"fmt"
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

const size = 1000
const numWorkers = 10

var state *core.BuildState

func TestBuildLotsOfTargets(t *testing.T) {
	config, _ := core.ReadConfigFiles(nil, "")
	state = core.NewBuildState(numWorkers, nil, 4, config)
	state.Parser = &fakeParser{
		PostBuildFunctions: buildFunctionMap{},
	}
	pkg := core.NewPackage("pkg")
	state.Graph.AddPackage(pkg)
	for i := 1; i <= size; i++ {
		addTarget(state, i)
	}
	var wg sync.WaitGroup
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func(i int) {
			please(i, state)
			wg.Done()
		}(i)
	}
	// Consume and discard any results
	go func() {
		for result := range state.Results {
			assert.NotEqual(t, core.TargetBuildFailed, result.Status)
			log.Info("%s", result.Description)
		}
	}()
	state.TaskDone(true) // Initial target adding counts as one.
	wg.Wait()
}

func addTarget(state *core.BuildState, i int) *core.BuildTarget {
	// Create and add a new target, with a parent and a dependency.
	target := core.NewBuildTarget(label(i))
	target.IsFilegroup = true // Will mean it doesn't have to shell out to anything.
	target.SetState(core.Active)
	state.Graph.AddTarget(target)
	if i <= size {
		if i > 10 {
			target.Flakiness = i // Stash this here, will be useful later.
			state.Parser.(*fakeParser).PostBuildFunctions[target] = postBuild
		}
		if i < size/10 {
			for j := 0; j < 10; j++ {
				dep := label(i*10 + j)
				log.Info("Adding dependency %s -> %s", target.Label, dep)
				target.AddDependency(dep)
				state.Graph.AddDependency(target.Label, dep)
			}
		} else {
			// These are buildable now
			state.AddPendingBuild(target.Label, false)
		}
	}
	state.AddActiveTarget()
	return target
}

func label(i int) core.BuildLabel {
	return core.ParseBuildLabel(fmt.Sprintf("//pkg:target%d", i), "")
}

// please mimics the core build 'loop' from src/please.go.
func please(tid int, state *core.BuildState) {
	for {
		label, _, t := state.NextTask()
		switch t {
		case core.Stop, core.Kill:
			return
		case core.Build:
			Build(tid, state, label)
		default:
			panic(fmt.Sprintf("unexpected task type: %d", t))
		}
		state.TaskDone(true)
	}
}

// Post-build function that adds new targets & ties in dependencies.
func postBuild(target *core.BuildTarget, out string) error {
	// Add a target corresponding to this one to its 'parent'
	if target.Flakiness == 0 {
		return fmt.Errorf("shouldn't be calling a post-build function on %s", target.Label)
	}
	parent := label(target.Flakiness / 10)
	newTarget := addTarget(state, target.Flakiness+size)

	// This mimics what interpreter.go does.
	state.Graph.TargetOrDie(parent).AddMaybeExportedDependency(newTarget.Label, false, false)
	state.Graph.AddDependency(parent, newTarget.Label)
	return nil
}

type buildFunctionMap map[*core.BuildTarget]func(*core.BuildTarget, string) error

type fakeParser struct {
	PostBuildFunctions buildFunctionMap
}

func (fake *fakeParser) ParseFile(state *core.BuildState, pkg *core.Package, filename string) error {
	return nil
}

func (fake *fakeParser) ParseReader(state *core.BuildState, pkg *core.Package, r io.ReadSeeker) error {
	return nil
}

func (fake *fakeParser) RunPreBuildFunction(threadID int, state *core.BuildState, target *core.BuildTarget) error {
	return nil
}

func (fake *fakeParser) RunPostBuildFunction(threadID int, state *core.BuildState, target *core.BuildTarget, output string) error {
	if f, present := fake.PostBuildFunctions[target]; present {
		return f(target, output)
	}
	return nil
}
