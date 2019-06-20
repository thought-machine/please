package core

import (
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
	priorityMask             = ^0x00FF
)

type pendingTask struct {
	Label    BuildLabel // Label of target to parse
	Dependor BuildLabel // The target that depended on it (only for parse tasks)
	Type     TaskType
}

func (t pendingTask) Compare(that queue.Item) int {
	return int((t.Type & priorityMask) - (that.(pendingTask).Type & priorityMask))
}

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
	// Stream of results pushed to remote clients.
	remoteResults chan *BuildResult
	// Last results for each thread. These are used to catch up remote clients quickly.
	lastResults []*BuildResult
	// Timestamp that the build is considered to start at.
	StartTime time.Time
	// Various system statistics. Mostly used during remote communication.
	Stats *SystemStats
	// Configuration options
	Config *Configuration
	// Parser implementation. Other things can call this to perform various external parse tasks.
	Parser Parser
	// Worker pool for the parser
	ParsePool Pool
	// Subprocess executor.
	ProcessExecutor *process.Executor
	// Hashes of variouts bits of the configuration, used for incrementality.
	Hashes struct {
		// Hash of the general config, not including specialised bits.
		Config []byte
	}
	// Tracks file hashes during the build.
	PathHasher *fs.PathHasher
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
	// The original architecture that the user requested to build for.
	OriginalArch cli.Arch
	// True if we require rule hashes to be correctly verified (usually the case).
	VerifyHashes bool
	// Aggregated coverage for this run
	Coverage TestCoverage
	// True if the build has been successful so far (i.e. nothing has failed yet).
	Success bool
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
	// True if we only need to parse the initial package (i.e. don't search downwards
	// through deps) - for example when doing `plz query print`.
	ParsePackageOnly bool
	// True if this build is triggered by watching for changes
	Watch bool
	// Number of times to run each test target. 1 == once each, plus flakes if necessary.
	NumTestRuns int
	// True to clean working directories after successful builds.
	CleanWorkdirs bool
	// True if we're forcing a rebuild of the original targets.
	ForceRebuild bool
	// True to always show test output, even on success.
	ShowTestOutput bool
	// True to print all output of all tasks to stderr.
	ShowAllOutput bool
	// True to attach a debugger on test failure.
	DebugTests bool
	// True if we think the underlying filesystem supports xattrs (which affects how we write some metadata).
	XattrsSupported bool
	// True once we have killed the workers, so we only do it once.
	workersKilled bool
	// Number of running workers
	numWorkers int
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
func (state *BuildState) AddPendingParse(label, dependor BuildLabel, forSubinclude bool) {
	atomic.AddInt64(&state.progress.numActive, 1)
	atomic.AddInt64(&state.progress.numPending, 1)
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
	if task.Type == Build || task.Type == SubincludeBuild || task.Type == Test {
		atomic.AddInt64(&state.progress.numRunning, 1)
	}
	return task.Label, task.Dependor, task.Type
}

func (state *BuildState) addPending(label BuildLabel, t TaskType) {
	atomic.AddInt64(&state.progress.numPending, 1)
	state.pendingTasks.Put(pendingTask{Label: label, Type: t})
}

// TaskDone indicates that a single task is finished. Should be called after one is finished with
// a task returned from NextTask(), or from a call to ExtraTask().
func (state *BuildState) TaskDone(wasBuildOrTest bool) {
	atomic.AddInt64(&state.progress.numDone, 1)
	if wasBuildOrTest {
		atomic.AddInt64(&state.progress.numRunning, -1)
	}
	if atomic.AddInt64(&state.progress.numPending, -1) <= 0 {
		state.Stop(state.numWorkers)
		state.killall(Stop)
	}
}

// Stop adds n stop tasks to the list of pending tasks, which stops n workers after all their other tasks are done.
func (state *BuildState) Stop(n int) {
	for i := 0; i < n; i++ {
		state.pendingTasks.Put(pendingTask{Type: Stop})
	}
}

// KillAll kills all the workers.
func (state *BuildState) KillAll() {
	state.killall(Kill)
}

func (state *BuildState) killall(signal TaskType) {
	if !state.workersKilled {
		state.workersKilled = true
		for i := 0; i < state.numWorkers; i++ {
			state.pendingTasks.Put(pendingTask{Type: signal})
		}
		if state.results != nil {
			close(state.results)
		}
		if state.remoteResults != nil {
			close(state.remoteResults)
		}
	}
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
		// The sets of original targets are duplicated between states for all architectures,
		// we must add it to all of them to ensure everything sees the same set.
		for _, s := range state.progress.allStates {
			s.OriginalTargets = append(s.OriginalTargets, label)
		}
	}
	state.AddPendingParse(label, OriginalTarget, false)
}

// LogBuildResult logs the result of a target either building or parsing.
func (state *BuildState) LogBuildResult(tid int, label BuildLabel, status BuildResultStatus, description string) {
	if status == PackageParsed {
		// We may have parse tasks waiting for this package to exist, check for them.
		state.progress.pendingPackageMutex.Lock()
		if ch, present := state.progress.pendingPackages[packageKey{Name: label.PackageName, Subrepo: label.Subrepo}]; present {
			close(ch) // This signals to anyone waiting that it's done.
		}
		state.progress.pendingPackageMutex.Unlock()
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
		// We may have parse tasks waiting for this guy to build, check for them.
		state.progress.pendingTargetMutex.Lock()
		if ch, present := state.progress.pendingTargets[label]; present {
			close(ch) // This signals to anyone waiting that it's done.
		}
		state.progress.pendingTargetMutex.Unlock()
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
			log.Notice("%s", r)
		}
	}()
	if state.results != nil {
		state.results <- result
	}
	if state.remoteResults != nil {
		state.remoteResults <- result
		state.lastResults[result.ThreadID] = result
	}
	if result.Status.IsFailure() {
		state.Success = false
	}
}

// Results returns a channel on which the caller can listen for results.
func (state *BuildState) Results() <-chan *BuildResult {
	if state.results == nil {
		state.results = make(chan *BuildResult, 100*state.numWorkers)
	}
	return state.results
}

// RemoteResults returns a channel for distributing remote results too, as well as
// the last set of results per thread.
func (state *BuildState) RemoteResults() (<-chan *BuildResult, []*BuildResult) {
	if state.remoteResults == nil {
		state.remoteResults = make(chan *BuildResult, 100*state.numWorkers)
	}
	return state.remoteResults, state.lastResults
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

// ExpandOriginalTargets expands any pseudo-targets (ie. :all, ... has already been resolved to a bunch :all targets)
// from the set of original targets.
// Deprecated: Callers should use ExpandOriginalLabels instead.
func (state *BuildState) ExpandOriginalTargets() BuildLabels {
	return state.ExpandOriginalLabels()
}

// ExpandOriginalLabels expands any pseudo-labels (ie. :all, ... has already been resolved to a bunch :all targets)
// from the set of original labels.
func (state *BuildState) ExpandOriginalLabels() BuildLabels {
	return state.ExpandLabels(state.OriginalTargets)
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
			if target.ShouldInclude(state.Include, state.Exclude) && (!state.NeedTests || target.IsTest) {
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
	for _, target := range state.ExpandOriginalTargets() {
		if !target.HasParent() || state.isOriginalTarget(target, true) {
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
		state.ParsePool.AddWorker()
		<-ch
		state.ParsePool.StopWorker()
		return state.Graph.PackageByLabel(label)
	}
	// Nothing's registered this so we do it ourselves.
	state.progress.pendingPackages[key] = make(chan struct{})
	state.progress.pendingPackageMutex.Unlock()
	return state.Graph.PackageByLabel(label) // Important to check again; it's possible to race against this whole lot.
}

// WaitForBuiltTarget blocks until the given label is available as a build target and has been successfully built.
func (state *BuildState) WaitForBuiltTarget(l, dependor BuildLabel) *BuildTarget {
	if t := state.Graph.Target(l); t != nil {
		if state := t.State(); state >= Built && state != Failed {
			return t
		}
	}
	dependor.Name = "all" // Every target in this package depends on this one.
	// okay, we need to register and wait for this guy.
	state.progress.pendingTargetMutex.Lock()
	if ch, present := state.progress.pendingTargets[l]; present {
		// Something's already registered for this, get on the train
		state.progress.pendingTargetMutex.Unlock()
		log.Debug("Pausing parse of %s to wait for %s", dependor, l)
		state.ParsePool.AddWorker()
		<-ch
		state.ParsePool.StopWorker()
		log.Debug("Resuming parse of %s now %s is ready", dependor, l)
		return state.Graph.Target(l)
	}
	// Nothing's registered this, set it up.
	state.progress.pendingTargets[l] = make(chan struct{})
	state.progress.pendingTargetMutex.Unlock()
	state.QueueTarget(l, dependor, false, true)
	// Do this all over; the re-checking that happens here is actually fairly important to resolve
	// a potential race condition if the target was built between us checking earlier and registering
	// the channel just now.
	return state.WaitForBuiltTarget(l, dependor)
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

// QueueTarget adds a single target to the build queue.
func (state *BuildState) QueueTarget(label, dependor BuildLabel, rescan, forceBuild bool) {
	target := state.Graph.Target(label)
	if target == nil {
		// If the package isn't loaded yet, we need to queue a parse for it.
		if state.Graph.PackageByLabel(label) == nil {
			state.AddPendingParse(label, dependor, forceBuild)
			return
		}
		// Package is loaded but target doesn't exist in it. Check again to avoid nasty races.
		target = state.Graph.Target(label)
		if target == nil {
			log.Fatalf("Target %s (referenced by %s) doesn't exist\n", label, dependor)
		}
	}
	if target.State() >= Active && !rescan && !forceBuild {
		return // Target is already tagged to be built and likely on the queue.
	}
	// Only do this bit if we actually need to build the target
	if !target.SyncUpdateState(Inactive, Semiactive) && !rescan && !forceBuild {
		return
	}
	if state.NeedBuild || forceBuild {
		if target.SyncUpdateState(Semiactive, Active) {
			state.AddActiveTarget()
			if target.IsTest && state.NeedTests {
				state.AddActiveTarget() // Tests count twice if we're gonna run them.
			}
		}
	}
	// If this target has no deps, add it to the queue now, otherwise handle its deps.
	// Only add if we need to build targets (not if we're just parsing) but we might need it to parse...
	if target.State() == Active && state.Graph.AllDepsBuilt(target) {
		if target.SyncUpdateState(Active, Pending) {
			state.AddPendingBuild(label, dependor.IsAllTargets())
		}
		if !rescan {
			return
		}
	}
	for _, dep := range target.DeclaredDependencies() {
		// Check the require/provide stuff; we may need to add a different target.
		if len(target.Requires) > 0 {
			if depTarget := state.Graph.Target(dep); depTarget != nil && len(depTarget.Provides) > 0 {
				for _, provided := range depTarget.ProvideFor(target) {
					state.QueueTarget(provided, label, false, forceBuild)
				}
				continue
			}
		}
		state.QueueTarget(dep, label, false, forceBuild)
	}
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
func NewBuildState(numThreads int, cache Cache, verbosity int, config *Configuration) *BuildState {
	// Deliberately ignore the error here so we don't require the sandbox tool until it's needed.
	sandboxTool, _ := LookBuildPath(config.Build.PleaseSandboxTool, config)
	state := &BuildState{
		Graph:           NewGraph(),
		pendingTasks:    queue.NewPriorityQueue(10000, true), // big hint, why not
		lastResults:     make([]*BuildResult, numThreads),
		PathHasher:      fs.NewPathHasher(RepoRoot, config.Build.Xattrs),
		ProcessExecutor: process.New(sandboxTool),
		StartTime:       startTime,
		Config:          config,
		Verbosity:       verbosity,
		Cache:           cache,
		ParsePool:       NewPool(numThreads),
		VerifyHashes:    true,
		NeedBuild:       true,
		Success:         true,
		XattrsSupported: config.Build.Xattrs,
		Coverage:        TestCoverage{Files: map[string][]LineCoverage{}},
		OriginalArch:    cli.HostArch(),
		numWorkers:      numThreads,
		Stats:           &SystemStats{},
		progress: &stateProgress{
			numActive:       1, // One for the initial target adding on the main thread.
			numRunning:      1, // Similarly.
			numPending:      1,
			pendingTargets:  map[BuildLabel]chan struct{}{},
			pendingPackages: map[packageKey]chan struct{}{},
		},
	}
	state.progress.allStates = []*BuildState{state}
	state.Hashes.Config = config.Hash()
	config.Please.NumThreads = numThreads
	for _, exp := range config.Parse.ExperimentalDir {
		state.experimentalLabels = append(state.experimentalLabels, BuildLabel{PackageName: exp, Name: "..."})
	}
	return state
}

// NewDefaultBuildState creates a BuildState for the default configuration.
// This is useful for tests etc that don't need to customise anything about it.
func NewDefaultBuildState() *BuildState {
	return NewBuildState(1, nil, 4, DefaultConfiguration())
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
