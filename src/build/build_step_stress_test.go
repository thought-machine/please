// Stress test around the build step stuff, specifically trying to
// identify concurrent map read / writes.

package build

import (
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"core"
)

const size = 1000
const numWorkers = 10

func init() {
	postBuildFunc = postBuild
}

func TestBuildLotsOfTargets(t *testing.T) {
	config, _ := core.ReadConfigFiles(nil)
	state := core.NewBuildState(numWorkers, nil, 4, config)
	pkg := core.NewPackage("pkg")
	state.Graph.AddPackage(pkg)
	for i := 1; i <= size; i++ {
		addTarget(state, i)
	}
	var wg sync.WaitGroup
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			please(i, state)
			wg.Done()
		}()
	}
	// Consume and discard any results
	go func() {
		for result := range state.Results {
			assert.NotEqual(t, core.TargetBuildFailed, result.Status)
			log.Info("%s", result.Description)
		}
	}()
	state.TaskDone() // Initial target adding counts as one.
	wg.Wait()
}

func addTarget(state *core.BuildState, i int) *core.BuildTarget {
	// Create and add a new target, with a parent and a dependency.
	target := core.NewBuildTarget(label(i))
	target.Command = "__FILEGROUP__" // Will mean it doesn't have to shell out to anything.
	target.SetState(core.Active)
	state.Graph.AddTarget(target)
	if i <= size {
		if i > 10 {
			target.TestTimeout = i // Stash this here, will be useful later.
			target.PostBuildFunction = reflect.ValueOf(&postBuildFunc).Pointer()
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
		state.TaskDone()
	}
}

// Post-build function that adds new targets & ties in dependencies.
func postBuild(target *core.BuildTarget, out string) error {
	// Add a target corresponding to this one to its 'parent'
	if target.TestTimeout == 0 {
		return fmt.Errorf("shouldn't be calling a post-build function on %s", target.Label)
	}
	parent := label(target.TestTimeout / 10)
	newTarget := addTarget(core.State, target.TestTimeout+size)

	// This mimics what interpreter.go does.
	core.State.Graph.TargetOrDie(parent).AddMaybeExportedDependency(newTarget.Label, false)
	core.State.Graph.AddDependency(parent, newTarget.Label)
	return nil
}

// Don't ask.
var postBuildFunc func(*core.BuildTarget, string) error
