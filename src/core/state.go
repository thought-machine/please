package core

import (
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"io"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Workiva/go-datastructures/queue"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/process"
)

// startTime is as close as we can conveniently get to process start time.
var startTime = time.Now()

// A taskType identifies the kind of task returned from NextTask()
type taskType int

// The values here are fiddled to make Compare work easily.
// Essentially we prioritise on the higher bits only and use the lower ones to make
// the values unique.
// Subinclude tasks order first, but we're happy for all build / parse / test tasks
// to be treated equivalently.
const (
	Kill            taskType = 0x0000 | 0 //nolint:staticcheck
	SubincludeBuild          = 0x1000 | 1
	SubincludeParse          = 0x2000 | 2
	Build                    = 0x4000 | 3
	Parse                    = 0x4000 | 4
	Test                     = 0x4000 | 5
	Stop                     = 0x8000 | 6
	priorityMask             = ^0x00FF
)

type pendingTask struct {
	Label     BuildLabel // Label of target to parse
	Dependent BuildLabel // The target that depended on it (only for parse tasks)
	Run       int        // The run number of this task (only for tests)
	Type      taskType
}

func (t pendingTask) Compare(that queue.Item) int {
	return int((t.Type & priorityMask) - (that.(pendingTask).Type & priorityMask))
}

// ParseTask is the type for the parse task queue
type ParseTask struct {
	Label, Dependent BuildLabel
	ForSubinclude    bool
}

// BuildTask is the type for the build task queue
type BuildTask = BuildLabel

// TestTask is the type for the test task queue
type TestTask struct {
	Label BuildLabel
	Run   int
}

// ParseTaskQueue is a channel to send parse tasks down to parse worker pool
type ParseTaskQueue = chan ParseTask

// BuildTaskQueue is a channel to send build tasks down to please worker pool
type BuildTaskQueue = chan BuildTask

// TestTaskQueue is a channel to send test tasks down to please worker pool
type TestTaskQueue = chan TestTask

// A Parser is the interface to reading and interacting with BUILD files.
type Parser interface {
	// ParseFile parses a single BUILD file into the given package.
	ParseFile(state *BuildState, pkg *Package, filename string) error
	// ParseReader parses a single BUILD file into the given package.
	ParseReader(state *BuildState, pkg *Package, reader io.ReadSeeker) error
	// RunPreBuildFunction runs a pre-build function for a target.
	RunPreBuildFunction(threadID int, state *BuildState, target *BuildTarget) error
	// RunPostBuildFunction runs a post-build function for a target.
	RunPostBuildFunction(threadID int, state *BuildState, target *BuildTarget, output string) error
}

// A RemoteClient is the interface to a remote execution service.
type RemoteClient interface {
	// Build invokes a build of the target remotely.
	Build(tid int, target *BuildTarget) (*BuildMetadata, error)
	// Test invokes a test run of the target remotely.
	Test(tid int, target *BuildTarget, run int) (metadata *BuildMetadata, err error)
	// Run executes the target remotely.
	Run(target *BuildTarget) error
	// Download downloads the outputs for the given target that has already been built remotely.
	Download(target *BuildTarget) error
	// PrintHashes shows the hashes of a target.
	PrintHashes(target *BuildTarget, isTest bool)
	// DataRate returns an estimate of the current in/out RPC data rates and totals so far in bytes per second.
	DataRate() (int, int, int, int)
}

// A TargetHasher is a thing that knows how to create hashes for targets.
type TargetHasher interface {
	// OutputHash calculates the output hash for a given build target.
	OutputHash(target *BuildTarget) ([]byte, error)
	// SetHash sets the output hash for a given build target.
	SetHash(target *BuildTarget, hash []byte)
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
	results chan *BuildResult
	// Timestamp that the build is considered to start at.
	StartTime time.Time
	// Various system statistics. Mostly used during remote communication.
	Stats *SystemStats
	// Configuration options
	Config *Configuration
	// Parser implementation. Other things can call this to perform various external parse tasks.
	Parser Parser
	// Subprocess executor.
	ProcessExecutor *process.Executor
	// Hashes of variouts bits of the configuration, used for incrementality.
	Hashes struct {
		// Hash of the general config, not including specialised bits.
		Config []byte
	}
	// Tracks file hashes during the build.
	PathHasher *fs.PathHasher
	// Hashers of all supported functions
	hashers map[string]*fs.PathHasher
	// Cache to store / retrieve old build results.
	Cache Cache
	// Client to remote execution service, if configured.
	RemoteClient RemoteClient
	// Hasher for targets
	TargetHasher TargetHasher
	// Arguments to tests.
	TestArgs []string
	// Labels of targets that we will include / exclude
	Include, Exclude []string
	// Actual targets to exclude from discovery
	ExcludeTargets []BuildLabel
	// The original architecture that the user requested to build for.
	OriginalArch cli.Arch
	// True if we require rule hashes to be correctly verified (usually the case).
	VerifyHashes bool
	// Aggregated coverage for this run
	Coverage TestCoverage
	// True if >= 1 target has failed to build
	BuildFailed bool
	// True if >= 1 target has failed test cases
	TestFailed bool
	// True if tests should calculate coverage metrics
	NeedCoverage bool
	// True if we intend to build targets. False if we're just parsing
	// (although some may be built if they're needed for parse).
	NeedBuild bool
	// True if we're running tests. False if we're only building or parsing.
	NeedTests bool
	// True if we will run targets at the end of the build.
	NeedRun bool
	// True if we want to calculate target hashes (ie. 'plz hash').
	NeedHashesOnly bool
	// True if we only want to prepare build directories (ie. 'plz build --prepare')
	PrepareOnly bool
	// True if we're going to run a shell after builds are prepared.
	PrepareShell bool
	// True if we will download outputs during remote execution.
	DownloadOutputs bool
	// True if we only need to parse the initial package (i.e. don't search downwards
	// through deps) - for example when doing `plz query print`.
	ParsePackageOnly bool
	// True if this build is triggered by watching for changes
	Watch bool
	// Number of times to run each test target. 1 == once each, plus flakes if necessary.
	NumTestRuns int
	// Whether to run multiple test runs sequentially or across multiple workers (can be useful if tests bind to ports
	// or similar)
	TestSequentially bool
	// True to clean working directories after successful builds.
	CleanWorkdirs bool
	// True if we're forcing a rebuild of the original targets.
	ForceRebuild bool
	// True if we're forcing to rerun tests of the targets.
	ForceRerun bool
	// True to always show test output, even on success.
	ShowTestOutput bool
	// True to print all output of all tasks to stderr.
	ShowAllOutput bool
	// True to attach a debugger on test failure.
	DebugTests bool
	// True if we think the underlying filesystem supports xattrs (which affects how we write some metadata).
	XattrsSupported bool
	// Experimental directories
	experimentalLabels []BuildLabel
	// Various items for tracking progress.
	progress *stateProgress
}

// A stateProgress records various points of progress for a State.
// This is split out from above so we can share it between multiple instances.
type stateProgress struct {
	// Used to count the number of currently active/pending targets
	numActive  int64
	numPending int64
	numRunning int64
	numDone    int64
	mutex      sync.Mutex
	// Used to track subinclude() calls that block until targets are built.
	pendingTargets     map[BuildLabel]chan struct{}
	pendingTargetMutex sync.Mutex
	// Used to track general package parsing requests.
	pendingPackages     map[packageKey]chan struct{}
	pendingPackageMutex sync.Mutex
	// The set of known states
	allStates []*BuildState
	// Targets that we were originally requested to build
	originalTargets     []BuildLabel
	originalTargetMutex sync.Mutex
	// True if the build has been successful so far (i.e. nothing has failed yet).
	success bool
}

// SystemStats stores information about the system.
type SystemStats struct {
	Memory struct {
		Total, Used uint64
		UsedPercent float64
	}
	// This is somewhat abbreviated to the "interesting" values and is aggregated
	// across all CPUs for convenience of display.
	// We're a bit casual about the exact definition of Used (it's the sum of some
	// fields that seem relevant) and IOWait is not fully reliable.
	CPU struct {
		Used, IOWait float64
		Count        int
	}
	// Number of currently running worker processes.
	// N.B. These are background workers as started by $(worker) commands, not the internal worker
	//      threads tracked in the state object.
	NumWorkerProcesses int
}

// AddActiveTarget increments the counter for a newly active build target.
func (state *BuildState) AddActiveTarget() {
	atomic.AddInt64(&state.progress.numActive, 1)
}

// AddPendingParse adds a task for a pending parse of a build label.
func (state *BuildState) AddPendingParse(label, dependent BuildLabel, forSubinclude bool) {
	atomic.AddInt64(&state.progress.numActive, 1)
	atomic.AddInt64(&state.progress.numPending, 1)
	if forSubinclude {
		state.pendingTasks.Put(pendingTask{Label: label, Dependent: dependent, Type: SubincludeParse})
	} else {
		state.pendingTasks.Put(pendingTask{Label: label, Dependent: dependent, Type: Parse})
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
func (state *BuildState) AddPendingTest(label BuildLabel, run int) {
	if state.NeedTests {
		state.addPendingTask(pendingTask{
			Label: label,
			Type:  Test,
			Run:   run,
		})
	}
}

// TaskQueues returns a set of channels to listen on for tasks of various types.
// This should only be called once per state (otherwise you will not get a full set of tasks).
func (state *BuildState) TaskQueues() (parses ParseTaskQueue, builds BuildTaskQueue, tests TestTaskQueue, remoteBuilds BuildTaskQueue, remoteTests TestTaskQueue) {
	p := make(chan ParseTask, 100)
	b := make(chan BuildTask, 100)
	t := make(chan TestTask, 100)
	rb := make(chan BuildLabel, 100)
	rt := make(chan TestTask, 100)
	go state.feedQueues(p, b, t, rb, rt)
	return p, b, t, rb, rt
}

// feedQueues feeds the build queues created in TaskQueues.
// We retain the internal priority queue since it is unbounded size which is pretty important
// for us not to deadlock.
func (state *BuildState) feedQueues(parses ParseTaskQueue, builds BuildTaskQueue, tests TestTaskQueue, remoteBuilds BuildTaskQueue, remoteTests TestTaskQueue) {
	anyRemote := state.Config.NumRemoteExecutors() > 0
	for {
		t, _ := state.pendingTasks.Get(1)
		task := t[0].(pendingTask)
		remote := func() bool {
			return anyRemote && !state.Graph.Target(task.Label).Local
		}

		switch task.Type {
		case Stop, Kill:
			close(parses)
			close(builds)
			close(tests)
			close(remoteBuilds)
			close(remoteTests)
			return
		case Parse, SubincludeParse:
			parses <- ParseTask{Label: task.Label, Dependent: task.Dependent, ForSubinclude: task.Type == SubincludeParse}
		case Build, SubincludeBuild:
			atomic.AddInt64(&state.progress.numRunning, 1)
			if remote() {
				remoteBuilds <- task.Label
			} else {
				builds <- task.Label
			}
		case Test:
			atomic.AddInt64(&state.progress.numRunning, 1)
			testTask := TestTask{
				Label: task.Label,
				Run:   task.Run,
			}
			if remote() {
				remoteTests <- testTask
			} else {
				tests <- testTask
			}
		}
	}
}

func (state *BuildState) addPending(label BuildLabel, t taskType) {
	state.addPendingTask(pendingTask{Label: label, Type: t})
}

func (state *BuildState) addPendingTask(task pendingTask) {
	atomic.AddInt64(&state.progress.numPending, 1)
	_ = state.pendingTasks.Put(task)
}

// TaskDone indicates that a single task is finished. Should be called after one is finished with
// a task returned from NextTask(), or from a call to ExtraTask().
func (state *BuildState) TaskDone(wasBuildOrTest bool) {
	atomic.AddInt64(&state.progress.numDone, 1)
	if wasBuildOrTest {
		atomic.AddInt64(&state.progress.numRunning, -1)
	}
	if atomic.AddInt64(&state.progress.numPending, -1) <= 0 {
		state.Stop()
	}
}

// Stop stops the worker queues after any current tasks are done.
func (state *BuildState) Stop() {
	state.pendingTasks.Put(pendingTask{Type: Stop})
}

// KillAll kills all the workers & closes the result channels.
func (state *BuildState) KillAll() {
	state.pendingTasks.Put(pendingTask{Type: Kill})
	state.CloseResults()
}

// CloseResults closes the result channels.
func (state *BuildState) CloseResults() {
	if state.results != nil {
		close(state.results)
	}
}

// IsOriginalTarget returns true if a target is an original target, ie. one specified on the command line.
func (state *BuildState) IsOriginalTarget(target *BuildTarget) bool {
	return state.isOriginalTarget(target, false)
}

func (state *BuildState) isOriginalTarget(target *BuildTarget, exact bool) bool {
	state.progress.originalTargetMutex.Lock()
	defer state.progress.originalTargetMutex.Unlock()
	for _, original := range state.progress.originalTargets {
		if original == target.Label || (!exact && original.IsAllTargets() && original.PackageName == target.Label.PackageName && state.ShouldInclude(target)) {
			return true
		}
	}
	return false
}

// IsOriginalTargetOrParent is like IsOriginalTarget but checks the target's parent too (if it has one)
func (state *BuildState) IsOriginalTargetOrParent(target *BuildTarget) bool {
	if state.IsOriginalTarget(target) {
		return true
	} else if parent := target.Parent(state.Graph); parent != nil {
		return state.IsOriginalTarget(parent)
	}
	return false
}

// SetIncludeAndExclude sets the include / exclude labels.
// Handles build labels on Exclude so should be preferred over setting them directly.
func (state *BuildState) SetIncludeAndExclude(include, exclude []string) {
	state.Include = include
	state.Exclude = nil
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

// ShouldInclude returns true if the given target is included by the include/exclude flags.
func (state *BuildState) ShouldInclude(target *BuildTarget) bool {
	for _, e := range state.ExcludeTargets {
		if e.Includes(target.Label) {
			return false
		}
	}
	return target.ShouldInclude(state.Include, state.Exclude)
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
		state.progress.originalTargetMutex.Lock()
		state.progress.originalTargets = append(state.progress.originalTargets, label)
		state.progress.originalTargetMutex.Unlock()
	}
	state.AddPendingParse(label, OriginalTarget, false)
}

// Hasher returns a PathHasher for the given function (e.g. "SHA1").
func (state *BuildState) Hasher(name string) *fs.PathHasher {
	hasher, present := state.hashers[name]
	if !present {
		log.Fatalf("Unknown hash type %s", name)
	}
	return hasher
}

// LogBuildResult logs the result of a target either building or parsing.
func (state *BuildState) LogBuildResult(tid int, label BuildLabel, status BuildResultStatus, description string) {
	if status == PackageParsed {
		func() {
			// We may have parse tasks waiting for this package to exist, check for them.
			state.progress.pendingPackageMutex.Lock()
			defer state.progress.pendingPackageMutex.Unlock()

			if ch, present := state.progress.pendingPackages[packageKey{Name: label.PackageName, Subrepo: label.Subrepo}]; present {
				close(ch) // This signals to anyone waiting that it's done.
			}
		}()
		return // We don't notify anything else on these.
	}
	state.LogResult(&BuildResult{
		ThreadID:    tid,
		Time:        time.Now(),
		Label:       label,
		Status:      status,
		Err:         nil,
		Description: description,
	})
	if status == TargetBuilt || status == TargetCached {
		func() {
			// We may have parse tasks waiting for this guy to build, check for them.
			state.progress.pendingTargetMutex.Lock()
			defer state.progress.pendingTargetMutex.Unlock()

			if ch, present := state.progress.pendingTargets[label]; present {
				close(ch) // This signals to anyone waiting that it's done.
			}
		}()
	}
}

// LogTestResult logs the result of a target once its tests have completed.
func (state *BuildState) LogTestResult(tid int, label BuildLabel, status BuildResultStatus, results *TestSuite, coverage *TestCoverage, err error, format string, args ...interface{}) {
	state.LogResult(&BuildResult{
		ThreadID:    tid,
		Time:        time.Now(),
		Label:       label,
		Status:      status,
		Err:         err,
		Description: fmt.Sprintf(format, args...),
		Tests:       *results,
	})
	state.progress.mutex.Lock()
	defer state.progress.mutex.Unlock()
	state.Coverage.Aggregate(coverage)
}

// LogBuildError logs a failure for a target to parse, build or test.
func (state *BuildState) LogBuildError(tid int, label BuildLabel, status BuildResultStatus, err error, format string, args ...interface{}) {
	state.LogResult(&BuildResult{
		ThreadID:    tid,
		Time:        time.Now(),
		Label:       label,
		Status:      status,
		Err:         err,
		Description: fmt.Sprintf(format, args...),
	})
}

// LogResult logs a build result directly to the state's queue.
func (state *BuildState) LogResult(result *BuildResult) {
	defer func() {
		if r := recover(); r != nil {
			// This is basically always "send on closed channel" which can happen because this
			// channel gets closed while there might still be some other workers doing stuff.
			// At that point we don't care much because the build has already failed.
			log.Info("%s", r)
		}
	}()
	if state.results != nil {
		state.results <- result
	}
	if result.Status.IsFailure() {
		state.progress.success = false
		if result.Status == TargetBuildFailed {
			state.BuildFailed = true
		} else if result.Status == TargetTestFailed {
			state.TestFailed = true
		}
	}
}

// Successful returns true if the state has been successful, i.e. no targets have errored.
func (state *BuildState) Successful() bool {
	return state.progress.success
}

// Results returns a channel on which the caller can listen for results.
func (state *BuildState) Results() <-chan *BuildResult {
	if state.results == nil {
		state.results = make(chan *BuildResult, 1000)
	}
	return state.results
}

// NumActive returns the number of currently active tasks (i.e. those that are
// scheduled to be built at some point, or have been built already).
func (state *BuildState) NumActive() int {
	return int(atomic.LoadInt64(&state.progress.numActive))
}

// NumDone returns the number of tasks that have been completed so far.
func (state *BuildState) NumDone() int {
	return int(atomic.LoadInt64(&state.progress.numDone))
}

// SetTaskNumbers allows a caller to set the number of active and done tasks.
// This may drastically confuse matters if used incorrectly.
func (state *BuildState) SetTaskNumbers(active, done int64) {
	atomic.StoreInt64(&state.progress.numActive, active)
	atomic.StoreInt64(&state.progress.numDone, done)
}

// ExpandOriginalLabels expands any pseudo-labels (ie. :all, ... has already been resolved to a bunch :all targets)
// from the set of original labels.
func (state *BuildState) ExpandOriginalLabels() BuildLabels {
	state.progress.originalTargetMutex.Lock()
	targets := state.progress.originalTargets[:]
	state.progress.originalTargetMutex.Unlock()
	return state.ExpandLabels(targets)
}

// ExpandLabels expands any pseudo-labels (ie. :all, ... has already been resolved to a bunch :all targets) from a set of labels.
func (state *BuildState) ExpandLabels(labels []BuildLabel) BuildLabels {
	ret := BuildLabels{}
	for _, label := range labels {
		if label.IsAllTargets() || label.IsAllSubpackages() {
			ret = append(ret, state.expandOriginalPseudoTarget(label)...)
		} else {
			ret = append(ret, label)
		}
	}
	return ret
}

// expandOriginalPseudoTarget expands one original pseudo-target (i.e. :all or /...) and sorts it
func (state *BuildState) expandOriginalPseudoTarget(label BuildLabel) BuildLabels {
	ret := BuildLabels{}
	addPackage := func(pkg *Package) {
		for _, target := range pkg.AllTargets() {
			if state.ShouldInclude(target) && (!state.NeedTests || target.IsTest) {
				ret = append(ret, target.Label)
			}
		}
	}
	if label.IsAllTargets() {
		if pkg := state.Graph.PackageByLabel(label); pkg != nil {
			addPackage(pkg)
		}
	} else {
		for name, pkg := range state.Graph.PackageMap() {
			if label.Includes(BuildLabel{PackageName: name}) {
				addPackage(pkg)
			}
		}
	}
	sort.Sort(ret)
	return ret
}

// ExpandVisibleOriginalTargets expands any pseudo-targets (ie. :all, ... has already been resolved to a bunch :all targets)
// from the set of original targets. Hidden targets are not included.
func (state *BuildState) ExpandVisibleOriginalTargets() BuildLabels {
	ret := BuildLabels{}
	for _, target := range state.ExpandOriginalLabels() {
		if !target.HasParent() || state.isOriginalTarget(state.Graph.TargetOrDie(target), true) {
			ret = append(ret, target)
		}
	}
	return ret
}

// WaitForPackage either returns the given package which is already parsed and available,
// or returns nil if nothing's parsed it already, in which case everything else calling this
// will wait for the caller to parse it themselves.
func (state *BuildState) WaitForPackage(label BuildLabel) *Package {
	if p := state.Graph.PackageByLabel(label); p != nil {
		return p
	}
	key := packageKey{Name: label.PackageName, Subrepo: label.Subrepo}
	state.progress.pendingPackageMutex.Lock()
	if ch, present := state.progress.pendingPackages[key]; present {
		state.progress.pendingPackageMutex.Unlock()
		<-ch
		return state.Graph.PackageByLabel(label)
	}
	// Nothing's registered this so we do it ourselves.
	state.progress.pendingPackages[key] = make(chan struct{})
	state.progress.pendingPackageMutex.Unlock()
	return state.Graph.PackageByLabel(label) // Important to check again; it's possible to race against this whole lot.
}

// WaitForBuiltTarget blocks until the given label is available as a build target and has been successfully built.
func (state *BuildState) WaitForBuiltTarget(l, dependent BuildLabel) *BuildTarget {
	if t := state.Graph.Target(l); t != nil {
		if s := t.State(); s >= Built && s != Failed {
			// Ensure we have downloaded its outputs if needed.
			// This is a bit fiddly but works around the case where we already built it but
			// didn't download, and now have found we need to.
			state.ensureDownloaded(t)
			return t
		}
	}
	dependent.Name = "all" // Every target in this package depends on this one.
	// okay, we need to register and wait for this guy.
	state.progress.pendingTargetMutex.Lock()
	if ch, present := state.progress.pendingTargets[l]; present {
		// Something's already registered for this, get on the train
		state.progress.pendingTargetMutex.Unlock()
		<-ch
		t := state.Graph.Target(l)
		state.ensureDownloaded(t)
		return t
	}
	// Nothing's registered this, set it up.
	state.progress.pendingTargets[l] = make(chan struct{})
	state.progress.pendingTargetMutex.Unlock()
	if err := state.QueueTarget(l, dependent, false, true); err != nil {
		log.Fatalf("%v", err)
	}

	// Do this all over; the re-checking that happens here is actually fairly important to resolve
	// a potential race condition if the target was built between us checking earlier and registering
	// the channel just now.
	return state.WaitForBuiltTarget(l, dependent)
}

// AddTarget adds a new target to the build graph.
func (state *BuildState) AddTarget(pkg *Package, target *BuildTarget) {
	pkg.AddTarget(target)
	state.Graph.AddTarget(target)
	if target.IsFilegroup {
		// At least register these guys as outputs.
		// It's difficult to handle non-file sources because we don't know if they're
		// parsed yet - recall filegroups are a special case for this since they don't
		// explicitly declare their outputs but can re-output other rules' outputs.
		for _, src := range target.AllLocalSources() {
			pkg.MustRegisterOutput(src, target)
		}
	} else {
		for _, out := range target.DeclaredOutputs() {
			pkg.MustRegisterOutput(out, target)
		}
		for _, out := range target.TestOutputs {
			if !fs.IsGlob(out) {
				pkg.MustRegisterOutput(out, target)
			}
		}
	}
}

// ShouldDownload returns true if the given target should be downloaded during remote execution.
func (state *BuildState) ShouldDownload(target *BuildTarget) bool {
	// Need to download the target if it was originally requested (and the user didn't pass --nodownload).
	// Also anything needed for subinclude needs to be local.
	return (state.IsOriginalTarget(target) && state.DownloadOutputs && !state.NeedTests) || target.NeededForSubinclude
}

// ShouldRebuild returns true if we should force a rebuild of this target (i.e. the user
// has done plz build --rebuild where we would not otherwise build it).
func (state *BuildState) ShouldRebuild(target *BuildTarget) bool {
	return state.ForceRebuild && state.IsOriginalTargetOrParent(target)
}

// WillRunRemotely returns true if the given target will be run on a remote executor.
func (state *BuildState) WillRunRemotely(target *BuildTarget) bool {
	return state.RemoteClient != nil && state.Config.NumRemoteExecutors() > 0 && !target.Local
}

// ensureDownloaded ensures that a target has been downloaded when built remotely.
// If remote execution is not enabled it has no effect.
func (state *BuildState) ensureDownloaded(target *BuildTarget) {
	if state.RemoteClient != nil {
		if err := state.RemoteClient.Download(target); err != nil {
			// Panicking is a bit crap but the places we are called from do not return an error.
			panic(fmt.Sprintf("Failed to download outputs for %s: %s", target, err))
		}
	}
}

// QueueTarget adds a single target to the build queue.
func (state *BuildState) QueueTarget(label, dependent BuildLabel, rescan, forceBuild bool) error {
	return state.queueTarget(label, dependent, rescan, forceBuild, forceBuild)
}

func (state *BuildState) queueTarget(label, dependent BuildLabel, rescan, forceBuild, neededForSubinclude bool) error {
	target := state.Graph.Target(label)
	if target == nil {
		// If the package isn't loaded yet, we need to queue a parse for it.
		if state.Graph.PackageByLabel(label) == nil {
			state.AddPendingParse(label, dependent, forceBuild)
			return nil
		}
		// Package is loaded but target doesn't exist in it. Check again to avoid nasty races.
		target = state.Graph.Target(label)
		if target == nil {
			return fmt.Errorf("Target %s (referenced by %s) doesn't exist", label, dependent)
		}
	}
	if target.State() >= Active && !rescan && !forceBuild {
		return nil // Target is already tagged to be built and likely on the queue.
	}
	// Only do this bit if we actually need to build the target
	if !target.SyncUpdateState(Inactive, Semiactive) && !rescan && !forceBuild {
		return nil
	}
	target.NeededForSubinclude = target.NeededForSubinclude || neededForSubinclude
	if state.NeedBuild || forceBuild {
		if target.SyncUpdateState(Semiactive, Active) {
			state.AddActiveTarget()
			if target.IsTest && state.NeedTests {
				if state.TestSequentially {
					atomic.AddInt64(&state.progress.numActive, 1)
				} else {
					// Tests count however many times we're going to run them if parallel.
					atomic.AddInt64(&state.progress.numActive, int64(state.NumTestRuns))
				}
			}
		}
	}
	// If this target has no deps, add it to the queue now, otherwise handle its deps.
	// Only add if we need to build targets (not if we're just parsing) but we might need it to parse...
	if target.State() == Active && state.Graph.AllDepsBuilt(target) {
		if target.SyncUpdateState(Active, Pending) {
			state.AddPendingBuild(label, dependent.IsAllTargets())
		}
		if !rescan {
			return nil
		}
	}
	for _, dep := range target.DeclaredDependencies() {
		// Check the require/provide stuff; we may need to add a different target.
		if len(target.Requires) > 0 {
			if depTarget := state.Graph.Target(dep); depTarget != nil && len(depTarget.Provides) > 0 {
				for _, provided := range depTarget.ProvideFor(target) {
					if err := state.queueTarget(provided, label, false, forceBuild, false); err != nil {
						return err
					}
				}
				continue
			}
		}
		if err := state.queueTarget(dep, label, false, forceBuild, false); err != nil {
			return err
		}
	}
	return nil
}

// ForTarget returns the state associated with a given target.
// This differs if the target is in a subrepo for a different architecture.
func (state *BuildState) ForTarget(target *BuildTarget) *BuildState {
	if target.Subrepo != nil && target.Subrepo.State != nil {
		return target.Subrepo.State
	}
	return state
}

// ForArch creates a copy of this BuildState for a different architecture.
func (state *BuildState) ForArch(arch cli.Arch) *BuildState {
	// Check if we've got this one already.
	// N.B. This implicitly handles the case of the host architecture
	if s := state.findArch(arch); s != nil {
		return s
	}
	// Copy with the architecture-specific config file.
	// This is slightly wrong in that other things (e.g. user-specified command line overrides) should
	// in fact take priority over this, but that's a lot more fiddly to get right.
	s := state.ForConfig(".plzconfig_" + arch.String())
	s.Config.Build.Arch = arch
	return s
}

// findArch returns an existing state for the given architecture, if one exists.
func (state *BuildState) findArch(arch cli.Arch) *BuildState {
	state.progress.mutex.Lock()
	defer state.progress.mutex.Unlock()
	for _, s := range state.progress.allStates {
		if s.Config.Build.Arch == arch {
			return s
		}
	}
	return nil
}

// ForConfig creates a copy of this BuildState based on the given config files.
func (state *BuildState) ForConfig(config ...string) *BuildState {
	state.progress.mutex.Lock()
	defer state.progress.mutex.Unlock()
	// Duplicate & alter configuration
	c := &Configuration{}
	*c = *state.Config
	c.buildEnvStored = &storedBuildEnv{}
	for _, filename := range config {
		if err := readConfigFile(c, filename); err != nil {
			log.Fatalf("Failed to read config file %s: %s", filename, err)
		}
	}
	s := &BuildState{}
	*s = *state
	s.Config = c
	state.progress.allStates = append(state.progress.allStates, s)
	return s
}

// DisableXattrs disables xattr support for this build. This is done for filesystems that
// don't support it.
func (state *BuildState) DisableXattrs() {
	state.XattrsSupported = false
	state.PathHasher.DisableXattrs()
}

// NewBuildState constructs and returns a new BuildState.
// Everyone should use this rather than attempting to construct it themselves;
// callers can't initialise all the required private fields.
func NewBuildState(config *Configuration) *BuildState {
	// Deliberately ignore the error here so we don't require the sandbox tool until it's needed.
	sandboxTool, _ := LookBuildPath(config.Build.PleaseSandboxTool, config)
	state := &BuildState{
		Graph:        NewGraph(),
		pendingTasks: queue.NewPriorityQueue(10000, true), // big hint, why not
		hashers: map[string]*fs.PathHasher{
			// For compatibility reasons the sha1 hasher has no suffix.
			"sha1":   fs.NewPathHasher(RepoRoot, config.Build.Xattrs, sha1.New, ""),
			"sha256": fs.NewPathHasher(RepoRoot, config.Build.Xattrs, sha256.New, "_sha256"),
		},
		ProcessExecutor: process.New(sandboxTool),
		StartTime:       startTime,
		Config:          config,
		VerifyHashes:    true,
		NeedBuild:       true,
		XattrsSupported: config.Build.Xattrs,
		Coverage:        TestCoverage{Files: map[string][]LineCoverage{}},
		OriginalArch:    cli.HostArch(),
		Stats:           &SystemStats{},
		progress: &stateProgress{
			numActive:       1, // One for the initial target adding on the main thread.
			numRunning:      1, // Similarly.
			numPending:      1,
			pendingTargets:  map[BuildLabel]chan struct{}{},
			pendingPackages: map[packageKey]chan struct{}{},
			success:         true,
		},
	}
	state.PathHasher = state.Hasher(config.Build.HashFunction)
	state.progress.allStates = []*BuildState{state}
	state.Hashes.Config = config.Hash()
	for _, exp := range config.Parse.ExperimentalDir {
		state.experimentalLabels = append(state.experimentalLabels, BuildLabel{PackageName: exp, Name: "..."})
	}
	return state
}

// NewDefaultBuildState creates a BuildState for the default configuration.
// This is useful for tests etc that don't need to customise anything about it.
func NewDefaultBuildState() *BuildState {
	return NewBuildState(DefaultConfiguration())
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
	Tests TestSuite
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
	TargetTestStopped
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
	case TargetTesting, TargetTestStopped, TargetTested, TargetTestFailed:
		return "Test"
	default:
		return "Other"
	}
}

// IsFailure returns true if this status represents a failure.
func (s BuildResultStatus) IsFailure() bool {
	return s == ParseFailed || s == TargetBuildFailed || s == TargetTestFailed
}

// IsActive returns true if this status represents a target that is not yet finished.
func (s BuildResultStatus) IsActive() bool {
	return s == PackageParsing || s == TargetBuilding || s == TargetTesting
}
