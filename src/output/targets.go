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
	FailedTargets  map[core.BuildLabel]error
	FailedNonTests []core.BuildLabel
}

func newBuildingTargets(state *core.BuildState, plainOutput bool) *buildingTargets {
	return &buildingTargets{
		plain:         plainOutput,
		state:         state,
		targets:       make([]buildingTarget, state.Config.Please.NumThreads+state.Config.NumRemoteExecutors()),
		FailedTargets: map[core.BuildLabel]error{},
	}
}

// Targets returns the set of currently known building targets, split into local and remote.
func (bt *buildingTargets) Targets() (local []buildingTarget, remote []buildingTarget) {
	n := bt.state.Config.Please.NumThreads
	return bt.targets[:n], bt.targets[n:]
}

// ProcessResult updates with a single result.
// It returns the label that was in this slot previously.
func (bt *buildingTargets) ProcessResult(result *core.BuildResult) core.BuildLabel {
	label := result.Label
	prev := bt.targets[result.ThreadID].Label
	if !result.Status.IsParse() { // Parse tasks happen on a different set of threads.
		if t := bt.state.Graph.Target(label); t != nil {
			bt.updateTarget(result, t)
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
	return prev
}

// updateTarget updates a single target with a new result.
func (bt *buildingTargets) updateTarget(result *core.BuildResult, t *core.BuildTarget) {
	target := &bt.targets[result.ThreadID]
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
