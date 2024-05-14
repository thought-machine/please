// Stress test around the build step stuff, specifically trying to
// identify concurrent map read / writes.

package build_test

import (
	"fmt"
	"io"
	iofs "io/fs"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/plz"
)

var log = logging.Log

const size = 1000

var state *core.BuildState

func TestBuildLotsOfTargets(t *testing.T) {
	config, _ := core.ReadConfigFiles(fs.HostFS, nil, nil)
	config.Please.NumThreads = 10
	state = core.NewBuildState(config)
	state.Parser = &fakeParser{
		PostBuildFunctions: buildFunctionMap{},
	}
	pkg := core.NewPackage("pkg")
	state.Graph.AddPackage(pkg)

	for i := 1; i <= size; i++ {
		addTarget(state, i)
	}
	state.TaskDone() // Initial target adding counts as one.

	results := state.Results()
	// Consume and discard any results
	go func() {
		for result := range results {
			assert.NotEqual(t, core.TargetBuildFailed, result.Status)
			log.Info("%s", result.Description)
		}
	}()

	plz.Run(nil, state)
}

func addTarget(state *core.BuildState, i int) *core.BuildTarget {
	// Create and add a new target, with a parent and a dependency.
	target := core.NewBuildTarget(label(i))
	target.IsFilegroup = true // Will mean it doesn't have to shell out to anything.
	target.SetState(core.Active)
	target.Test = new(core.TestFields)
	state.Graph.AddTarget(target)
	if i <= size {
		if i > 10 {
			target.Test.Flakiness = uint8(i) // Stash this here, will be useful later.
			state.Parser.(*fakeParser).PostBuildFunctions[target] = postBuild
		}
		if i < size/10 {
			for j := 0; j < 10; j++ {
				dep := label(i*10 + j)
				log.Info("Adding dependency %s -> %s", target.Label, dep)
				target.AddDependency(dep)
			}
		} else {
			// These are buildable now
			state.QueueTarget(target.Label, core.OriginalTarget, false, core.ParseModeNormal)
		}
	}
	return target
}

func label(i int) core.BuildLabel {
	return core.ParseBuildLabel(fmt.Sprintf("//pkg:target%d", i), "")
}

// Post-build function that adds new targets & ties in dependencies.
func postBuild(target *core.BuildTarget, out string) error {
	// Add a target corresponding to this one to its 'parent'
	if target.Test.Flakiness == 0 {
		return fmt.Errorf("shouldn't be calling a post-build function on %s", target.Label)
	}
	parent := label(int(target.Test.Flakiness / 10))
	newTarget := addTarget(state, int(target.Test.Flakiness)+size)

	// This mimics what interpreter.go does.
	t := state.Graph.TargetOrDie(parent)
	t.AddDependency(newTarget.Label)
	return nil
}

type buildFunctionMap map[*core.BuildTarget]func(*core.BuildTarget, string) error

type fakeParser struct {
	PostBuildFunctions buildFunctionMap
}

func (fake *fakeParser) RegisterPreload(core.BuildLabel) error {
	return nil
}

// ParseFile stub
func (fake *fakeParser) ParseFile(pkg *core.Package, label, dependent *core.BuildLabel, mode core.ParseMode, fs iofs.FS, filename string) error {
	return nil
}

func (fake *fakeParser) WaitForInit() {

}

// PreloadSubinclude stub
func (fake *fakeParser) NewParser(state *core.BuildState) {

}

// Init stub
func (fake *fakeParser) Init(state *core.BuildState) {

}

// ParseReader stub
func (fake *fakeParser) ParseReader(pkg *core.Package, r io.ReadSeeker, label, dependent *core.BuildLabel, mode core.ParseMode) error {
	return nil
}

// RunPreBuildFunction stub
func (fake *fakeParser) RunPreBuildFunction(state *core.BuildState, target *core.BuildTarget) error {
	return nil
}

// RunPostBuildFunction stub
func (fake *fakeParser) RunPostBuildFunction(state *core.BuildState, target *core.BuildTarget, output string) error {
	if f, present := fake.PostBuildFunctions[target]; present {
		return f(target, output)
	}
	return nil
}

// BuildRuleArgOrder stub
func (fake *fakeParser) BuildRuleArgOrder() map[string]int {
	return map[string]int{}
}
