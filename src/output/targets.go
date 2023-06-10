package output

import (
	"time"

	"github.com/thought-machine/please/src/core"
)

// Represents the current state of a single currently building target.
type buildingTarget struct {
	Label        core.BuildLabel
	Started      time.Time
	Finished     time.Time
	Description  string
	Err          error
	Colour       string
	Target       *core.BuildTarget
	Eta          time.Duration
	LastProgress float32
	BPS          float32
	Active       bool
	Failed       bool
	Cached       bool
}

// Collects all the currently building targets.
type buildingTargets struct {
	plain          bool
	state          *core.BuildState
	targets        []buildingTarget
	currentTargets map[core.BuildLabel]int
	available      []int
	FailedTargets  map[core.BuildLabel]error
	FailedNonTests []core.BuildLabel
}

func newBuildingTargets(state *core.BuildState, plainOutput bool) *buildingTargets {
	n := state.Config.Please.NumThreads + state.Config.NumRemoteExecutors()
	available := make([]int, n)
	for i := 0; i < n; i++ {
		available[i] = n - i // Do them backwards so the earliest indices are the first we'll take
	}
	return &buildingTargets{
		plain:          plainOutput,
		state:          state,
		targets:        make([]buildingTarget, n),
		currentTargets: make(map[core.BuildLabel]int, n),
		available:      available,
		FailedTargets:  map[core.BuildLabel]error{},
	}
}

// Targets returns the set of currently known building targets, split into local and remote.
func (bt *buildingTargets) Targets() (local []buildingTarget, remote []buildingTarget) {
	n := bt.state.Config.Please.NumThreads
	return bt.targets[:n], bt.targets[n:]
}

// ProcessResult updates with a single result.
// It returns the label that was in this slot previously and a 'thread id' for it (which is relevant for trace output)
func (bt *buildingTargets) ProcessResult(result *core.BuildResult) (core.BuildLabel, int) {
	label := result.Label
	idx := bt.index(label)
	prev := bt.targets[idx].Label
	if !result.Status.IsParse() { // Parse tasks aren't displayed here
		if t := bt.state.Graph.Target(label); t != nil {
			bt.updateTarget(idx, result, t)
		}
	}
	if result.Status.IsFailure() {
		bt.FailedTargets[label] = result.Err
		// Don't stop here after test failure, aggregate them for later.
		if result.Status != core.TargetTestFailed {
			// Reset colour so the entire compiler error output doesn't appear red.
			log.Errorf("%s failed:\x1b[0m\n%s", label, shortError(result.Err))
			bt.state.Stop()
		} else if msg := shortError(result.Err); msg != "" {
			log.Errorf("%s failed: %s", result.Label, msg)
		} else {
			log.Errorf("%s failed", label)
		}
		if result.Status != core.TargetTestFailed {
			bt.FailedNonTests = append(bt.FailedNonTests, label)
		}
	} else if result.Status == core.TargetBuildStopped {
		bt.FailedTargets[label] = nil
	} else if bt.plain && bt.state.ShowTestOutput && result.Status == core.TargetTested {
		// If using interactive output we'll print it afterwards.
		for _, testCase := range bt.state.Graph.TargetOrDie(label).Test.Results.TestCases {
			printf("Finished test %s:\n", testCase.Name)
			for _, testExecution := range testCase.Executions {
				showExecutionOutput(testExecution)
			}
		}
	}
	return prev, idx
}

// index returns the index to use for a result
func (bt *buildingTargets) index(label core.BuildLabel) int {
	if idx, present := bt.currentTargets[label]; present {
		return idx
	}
	// Grab whatever is available
	if len(bt.available) > 0 {
		n := len(bt.available) - 1
		idx := bt.available[n]
		bt.available = bt.available[:n]
		return idx
	}
	// Nothing available. This theoretically shouldn't happen - let's see in practice...
	return len(bt.currentTargets) - 1
}

// updateTarget updates a single target with a new result.
func (bt *buildingTargets) updateTarget(idx int, result *core.BuildResult, t *core.BuildTarget) {
	target := &bt.targets[idx]
	target.Label = result.Label
	target.Description = result.Description
	active := result.Status.IsActive()
	if !target.Active {
		// Starting to build now.
		target.Started = time.Now()
		target.Finished = target.Started
	} else if !active {
		// finished building
		target.Finished = time.Now()
	}
	target.Active = active
	target.Failed = result.Status.IsFailure()
	target.Cached = result.Status == core.TargetCached || result.Tests.Cached
	target.Err = result.Err
	target.Colour = targetColour(t)
	target.Target = t

	if bt.plain {
		if !active {
			active := pluralise(bt.state.NumActive(), "task", "tasks")
			log.Info("[%d/%s] %s: %s [%3.1fs]", bt.state.NumDone(), active, result.Label, result.Description, time.Since(target.Started).Seconds())
		} else {
			log.Info("%s: %s", result.Label, result.Description)
		}
	}
	if !active {
		bt.available = append(bt.available, idx)
		delete(bt.currentTargets, t.Label)
	} else {
		bt.currentTargets[t.Label] = idx
	}
}

func targetColour(target *core.BuildTarget) string {
	if target == nil {
		return "${BOLD_CYAN}" // unknown
	} else if target.IsBinary {
		return "${BOLD}" + targetColour2(target)
	} else {
		return targetColour2(target)
	}
}

func targetColour2(target *core.BuildTarget) string {
	for _, require := range target.Requires {
		if colour, present := replacements[require]; present {
			return colour
		}
	}
	return "${WHITE}"
}
