package core

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Workiva/go-datastructures/queue"
)

// A TaskType identifies the kind of task returned from NextTask()
type TaskType int

// The values here are fiddled to make Compare work easily.
// Essentially we prioritise on the higher bits only and use the lower ones to make
// the values unique.
// Subinclude tasks order first, but we're happy for all build / parse / test tasks
// to be treated equivalently.
const (
	Kill            TaskType = 0x0000 | 0
	SubincludeBuild          = 0x1000 | 1
	SubincludeParse          = 0x2000 | 2
	Build                    = 0x4000 | 3
	Parse                    = 0x4000 | 4
	Test                     = 0x4000 | 5
	Stop                     = 0x8000 | 6
	priorityMask             = ^0x0FFF
)

type pendingTask struct {
	Label    BuildLabel // Label of target to parse
	Dependor BuildLabel // The target that depended on it (only for parse tasks)
	Type     TaskType
}

func (t pendingTask) Compare(that queue.Item) int {
	return int((t.Type & priorityMask) - (that.(pendingTask).Type & priorityMask))
}

// A Parser allows performing several parsing tasks directly. This is not used for
// normal BUILD file parsing, but rather pre/post build callbacks etc to decouple
// the build package from calling straight into parse (since parse is cgo we attempt
// to minimise any dependencies on it).
type Parser interface {
	// RunPreBuildFunction runs a pre-build function for a target.
	RunPreBuildFunction(threadID int, state *BuildState, target *BuildTarget) error
	// RunPostBuildFunction runs a post-build function for a target.
	RunPostBuildFunction(threadID int, state *BuildState, target *BuildTarget, output string) error
	// UndeferAnyParses undefers any pending parses that are waiting for this target to build.
	UndeferAnyParses(state *BuildState, target *BuildTarget)
}

// A BuildState tracks the current state of the build & related data.
// As well as tracking the build graph and config, it also tracks the set of current
// tasks and maintains a queue of them, along with various related counters which are
// used to determine when we're finished.
// Tasks are internally tracked by priority, which is determined by their type.
type BuildState struct {
	Graph *BuildGraph
	// Stream of pending tasks
	pendingTasks *queue.PriorityQueue
	// Stream of results from the build
	Results chan *BuildResult
	// Configuration options
	Config *Configuration
	// Parser implementation. Other things can call this to perform various external parse tasks.
	Parser Parser
	// Hashes of variouts bits of the configuration, used for incrementality.
	Hashes struct {
		// Hash of the general config, not including specialised bits.
		Config []byte
		// Hash of the config relating to containerisation for tests.
		Containerisation []byte
	}
	// Level of verbosity during the build
	Verbosity int
	// Cache to store / retrieve old build results.
	Cache Cache
	// Targets that we were originally requested to build
	OriginalTargets []BuildLabel
	// Arguments to tests.
	TestArgs []string
	// Labels of targets that we will include / exclude
	Include, Exclude []string
	// Actual targets to exclude from discovery
	ExcludeTargets []BuildLabel
	// True if we require rule hashes to be correctly verified (usually the case).
	VerifyHashes bool
	// Aggregated coverage for this run
	Coverage TestCoverage
	// True if tests should calculate coverage metrics
	NeedCoverage bool
	// True if we intend to build targets. False if we're just parsing
	// (although some may be built if they're needed for parse).
	NeedBuild bool
	// True if we're running tests. False if we're only building or parsing.
	NeedTests bool
	// True if we want to calculate target hashes (ie. 'plz hash').
	NeedHashesOnly bool
	// True if we only want to prepare build directories (ie. 'plz build --prepare')
	PrepareOnly bool
	// True if we're going to run a shell after builds are prepared.
	PrepareShell bool
	// Number of times to run each test target. 0 == once each, plus flakes if necessary.
	NumTestRuns int
	// True to clean working directories after successful builds.
	CleanWorkdirs bool
	// True if we're forcing a rebuild of the original targets.
	ForceRebuild bool
	// True to always show test output, even on success.
	ShowTestOutput bool
	// True to print all output of all tasks to stderr.
	ShowAllOutput bool
	// Number of running workers
	numWorkers int
	// Experimental directory
	experimentalLabel BuildLabel
	// Used to count the number of currently active/pending targets
	numActive  int64
	numPending int64
	numDone    int64
	mutex      sync.Mutex
}

// State is a singleton instance of the most recently constructed BuildState.
// We would prefer not to have this but it ended up being inevitable for some of the
// parsing code that doesn't have ready access to other Go variables.
var State *BuildState

// AddActiveTarget increments the counter for a newly active build target.
func (state *BuildState) AddActiveTarget() {
	atomic.AddInt64(&state.numActive, 1)
}

// AddPendingParse adds a task for a pending parse of a build label.
func (state *BuildState) AddPendingParse(label, dependor BuildLabel, forSubinclude bool) {
	atomic.AddInt64(&state.numActive, 1)
	atomic.AddInt64(&state.numPending, 1)
	if forSubinclude {
		state.pendingTasks.Put(pendingTask{Label: label, Dependor: dependor, Type: SubincludeParse})
	} else {
		state.pendingTasks.Put(pendingTask{Label: label, Dependor: dependor, Type: Parse})
	}
}

// AddPendingBuild adds a task for a pending build of a target.
func (state *BuildState) AddPendingBuild(label BuildLabel, forSubinclude bool) {
	if forSubinclude {
		state.addPending(label, SubincludeBuild)
	} else {
		state.addPending(label, Build)
	}
}

// AddPendingTest adds a task for a pending test of a target.
func (state *BuildState) AddPendingTest(label BuildLabel) {
	if state.NeedTests {
		state.addPending(label, Test)
	}
}

// NextTask receives the next task that should be processed according to the priority queues.
func (state *BuildState) NextTask() (BuildLabel, BuildLabel, TaskType) {
	t, err := state.pendingTasks.Get(1)
	if err != nil {
		log.Fatalf("error receiving next task: %s", err)
	}
	task := t[0].(pendingTask)
	return task.Label, task.Dependor, task.Type
}

func (state *BuildState) addPending(label BuildLabel, t TaskType) {
	atomic.AddInt64(&state.numPending, 1)
	state.pendingTasks.Put(pendingTask{Label: label, Type: t})
}

// TaskDone indicates that a single task is finished. Should be called after one is finished with
// a task returned from NextTask().
func (state *BuildState) TaskDone() {
	atomic.AddInt64(&state.numDone, 1)
	if atomic.AddInt64(&state.numPending, -1) <= 0 {
		state.Stop(state.numWorkers)
	}
}

// Stop adds n stop tasks to the list of pending tasks, which stops n workers after all their other tasks are done.
func (state *BuildState) Stop(n int) {
	for i := 0; i < n; i++ {
		state.pendingTasks.Put(pendingTask{Type: Stop})
	}
}

// Kill adds n kill tasks to the list of pending tasks, which stops n workers before they do anything else.
func (state *BuildState) Kill(n int) {
	for i := 0; i < n; i++ {
		state.pendingTasks.Put(pendingTask{Type: Kill})
	}
}

// KillAll kills all the workers.
func (state *BuildState) KillAll() {
	state.Kill(state.numWorkers)
}

// IsOriginalTarget returns true if a target is an original target, ie. one specified on the command line.
func (state *BuildState) IsOriginalTarget(label BuildLabel) bool {
	return state.isOriginalTarget(label, false)
}

// isOriginalTarget implementsIsOriginalTarget, optionally allowing disabling matching :all labels.
func (state *BuildState) isOriginalTarget(label BuildLabel, exact bool) bool {
	for _, original := range state.OriginalTargets {
		if original == label || (!exact && original.IsAllTargets() && original.PackageName == label.PackageName) {
			return true
		}
	}
	return false
}

// SetIncludeAndExclude sets the include / exclude labels.
// Handles build labels on Exclude so should be preferred over setting them directly.
func (state *BuildState) SetIncludeAndExclude(include, exclude []string) {
	state.Include = include
	for _, e := range exclude {
		if LooksLikeABuildLabel(e) {
			if label, err := parseMaybeRelativeBuildLabel(e, ""); err != nil {
				log.Fatalf("%s", err)
			} else {
				state.ExcludeTargets = append(state.ExcludeTargets, label)
			}
		} else {
			state.Exclude = append(state.Exclude, e)
		}
	}
}

// AddOriginalTarget adds one of the original targets and enqueues it for parsing / building.
func (state *BuildState) AddOriginalTarget(label BuildLabel, addToList bool) {
	// Check it's not excluded first.
	for _, e := range state.ExcludeTargets {
		if e.Includes(label) {
			return
		}
	}
	if addToList {
		state.OriginalTargets = append(state.OriginalTargets, label)
	}
	state.AddPendingParse(label, OriginalTarget, false)
}

// LogBuildResult logs the result of a target either building or parsing.
func (state *BuildState) LogBuildResult(tid int, label BuildLabel, status BuildResultStatus, description string) {
	state.Results <- &BuildResult{
		ThreadID:    tid,
		Time:        time.Now(),
		Label:       label,
		Status:      status,
		Err:         nil,
		Description: description,
	}
}

// LogTestResult logs the result of a target once its tests have completed.
func (state *BuildState) LogTestResult(tid int, label BuildLabel, status BuildResultStatus, results *TestResults, coverage *TestCoverage, err error, format string, args ...interface{}) {
	state.Results <- &BuildResult{
		ThreadID:    tid,
		Time:        time.Now(),
		Label:       label,
		Status:      status,
		Err:         err,
		Description: fmt.Sprintf(format, args...),
		Tests:       *results,
	}
	state.mutex.Lock()
	defer state.mutex.Unlock()
	state.Coverage.Aggregate(coverage)
}

// LogBuildError logs a failure for a target to parse, build or test.
func (state *BuildState) LogBuildError(tid int, label BuildLabel, status BuildResultStatus, err error, format string, args ...interface{}) {
	state.Results <- &BuildResult{
		ThreadID:    tid,
		Time:        time.Now(),
		Label:       label,
		Status:      status,
		Err:         err,
		Description: fmt.Sprintf(format, args...),
	}
}

// NumActive returns the number of currently active tasks (i.e. those that are
// scheduled to be built at some point, or have been built already).
func (state *BuildState) NumActive() int {
	return int(atomic.LoadInt64(&state.numActive))
}

// NumDone returns the number of tasks that have been completed so far.
func (state *BuildState) NumDone() int {
	return int(atomic.LoadInt64(&state.numDone))
}

// ExpandOriginalTargets expands any pseudo-targets (ie. :all, ... has already been resolved to a bunch :all targets)
// from the set of original targets.
func (state *BuildState) ExpandOriginalTargets() BuildLabels {
	ret := BuildLabels{}
	addPackage := func(pkg *Package) {
		for _, target := range pkg.Targets {
			if target.ShouldInclude(state.Include, state.Exclude) && (!state.NeedTests || target.IsTest) {
				ret = append(ret, target.Label)
			}
		}
	}
	for _, label := range state.OriginalTargets {
		if label.IsAllTargets() {
			addPackage(state.Graph.PackageOrDie(label.PackageName))
		} else if label.IsAllSubpackages() {
			for name, pkg := range state.Graph.PackageMap() {
				if label.Includes(BuildLabel{PackageName: name}) {
					addPackage(pkg)
				}
			}
		} else {
			ret = append(ret, label)
		}
	}
	sort.Sort(ret)
	return ret
}

// ExpandVisibleOriginalTargets expands any pseudo-targets (ie. :all, ... has already been resolved to a bunch :all targets)
// from the set of original targets. Hidden targets are not included.
func (state *BuildState) ExpandVisibleOriginalTargets() BuildLabels {
	ret := BuildLabels{}
	for _, target := range state.ExpandOriginalTargets() {
		if !target.HasParent() || state.isOriginalTarget(target, true) {
			ret = append(ret, target)
		}
	}
	return ret
}

// NewBuildState constructs and returns a new BuildState.
// Everyone should use this rather than attempting to construct it themselves;
// callers can't initialise all the required private fields.
func NewBuildState(numThreads int, cache Cache, verbosity int, config *Configuration) *BuildState {
	State = &BuildState{
		Graph:             NewGraph(),
		pendingTasks:      queue.NewPriorityQueue(10000, true), // big hint, why not
		Results:           make(chan *BuildResult, numThreads*100),
		Config:            config,
		Verbosity:         verbosity,
		Cache:             cache,
		VerifyHashes:      true,
		NeedBuild:         true,
		numActive:         1, // One for the initial target adding on the main thread.
		numPending:        1,
		Coverage:          TestCoverage{Files: map[string][]LineCoverage{}},
		numWorkers:        numThreads,
		experimentalLabel: BuildLabel{PackageName: config.Please.ExperimentalDir, Name: "..."},
	}
	State.Hashes.Config = config.Hash()
	State.Hashes.Containerisation = config.ContainerisationHash()
	return State
}

// A BuildResult represents a single event in the build process, i.e. a target starting or finishing
// building, or reaching some milestone within those steps.
type BuildResult struct {
	// Thread id (or goroutine id, really) that generated this result.
	ThreadID int
	// Timestamp of this event
	Time time.Time
	// Target which has just changed
	Label BuildLabel
	// Its current status
	Status BuildResultStatus
	// Error, only populated for failure statuses
	Err error
	// Description of what's going on right now.
	Description string
	// Test results
	Tests TestResults
}

// A BuildResultStatus represents the status of a target when we log a build result.
type BuildResultStatus int

// The collection of expected build result statuses.
const (
	PackageParsing BuildResultStatus = iota
	PackageParsed
	ParseFailed
	TargetBuilding
	TargetBuildStopped
	TargetBuilt
	TargetCached
	TargetBuildFailed
	TargetTesting
	TargetTested
	TargetTestFailed
)

// Category returns the broad area that this event represents in the tasks we perform for a target.
func (s BuildResultStatus) Category() string {
	switch s {
	case PackageParsing, PackageParsed, ParseFailed:
		return "Parse"
	case TargetBuilding, TargetBuildStopped, TargetBuilt, TargetBuildFailed:
		return "Build"
	case TargetTesting, TargetTested, TargetTestFailed:
		return "Test"
	default:
		return "Other"
	}
}
