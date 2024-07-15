package core

import (
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"hash"
	"hash/crc32"
	"hash/crc64"
	"io"
	iofs "io/fs"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/zeebo/blake3"
	"golang.org/x/sync/errgroup"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/cmap"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/process"
)

type ParseMode uint8

const (
	ParseModeNormal ParseMode = 1 << iota
	ParseModeForSubinclude
	ParseModeForPreload
	ParseModeForceBuild
)

func (m ParseMode) IsPreload() bool {
	return m&ParseModeForPreload != 0
}

func (m ParseMode) IsForSubinclude() bool {
	return m&ParseModeForSubinclude != 0
}

// startTime is as close as we can conveniently get to process start time.
var startTime = time.Now()

// cycleCheckDuration is the length of time we allow inactivity for before we trigger cycle detection.
const cycleCheckDuration = 5 * time.Second

// ParseTask is the type for the parse task queue
type ParseTask struct {
	Label, Dependent BuildLabel
	Mode             ParseMode
}

// A TaskType identifies whether a task is a build or test action.
type TaskType uint8

const (
	BuildTask TaskType = 0
	TestTask  TaskType = 1
)

// A Task is the type for the queue of build/test tasks.
type Task struct {
	Target *BuildTarget
	Type   TaskType
	Run    uint32 // Only present for tests (the run of a build is always zero)
}

// A OutputDownloadOption is the option for how outputs should be downloaded.
type OutputDownloadOption uint8

const (
	// Don't download outputs.
	NoOutputDownload OutputDownloadOption = iota
	// Download original target outputs only.
	OriginalOutputDownload
	// Download original target's transitive outputs too.
	TransitiveOutputDownload
)

// A Parser is the interface to reading and interacting with BUILD files.
type Parser interface {
	// ParseFile parses a single BUILD file into the given package.
	ParseFile(pkg *Package, forLabel, dependent *BuildLabel, mode ParseMode, fs iofs.FS, filename string) error
	// ParseReader parses a single BUILD file into the given package.
	ParseReader(pkg *Package, reader io.ReadSeeker, forLabel, dependent *BuildLabel, mode ParseMode) error
	// RunPreBuildFunction runs a pre-build function for a target.
	RunPreBuildFunction(state *BuildState, target *BuildTarget) error
	// RunPostBuildFunction runs a post-build function for a target.
	RunPostBuildFunction(state *BuildState, target *BuildTarget, output string) error
	RegisterPreload(label BuildLabel) error
}

// A RemoteClient is the interface to a remote execution service.
type RemoteClient interface {
	// Build invokes a build of the target remotely.
	Build(target *BuildTarget) (*BuildMetadata, error)
	// Test invokes a test run of the target remotely.
	Test(target *BuildTarget, run int) (metadata *BuildMetadata, err error)
	// Run executes the target remotely.
	Run(target *BuildTarget) error
	// Download downloads the outputs for the given target that has already been built remotely.
	Download(target *BuildTarget) error
	// DownloadInputs downloads the whole of inputs folder for the given target that has already
	// been built remotely, into the target directory
	DownloadInputs(target *BuildTarget, targetDir string, isTest bool) error
	// PrintHashes shows the hashes of a target.
	PrintHashes(target *BuildTarget, isTest bool)
	// DataRate returns an estimate of the current in/out RPC data rates and totals so far in bytes per second.
	DataRate() (int, int, int, int)
	// Disconnect disconnects from the remote execution server.
	Disconnect() error
	// SubrepoFS returns a virtual filesystem for the subrepo target
	SubrepoFS(target *BuildTarget, root string) iofs.FS
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
	// Streams of pending tasks
	pendingParses  chan ParseTask
	pendingActions chan Task
	// Timestamp that the build is considered to start at.
	StartTime time.Time
	// Various system statistics. Mostly used during remote communication.
	stats *lockedStats
	// Configuration options
	Config *Configuration
	// The .plzconfig file for this repo. Unlike Config, no default values are applied. This will represent the
	// .plzconfig in a subrepo.
	RepoConfig *Configuration
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
	// The original architecture that the user requested to build for. Either the host arch, --arch, or the arch in the
	// .plzconfig
	TargetArch cli.Arch
	// The architecture this state is for. This might change as we re-parse packages for different architectures e.g.
	// for tools that run on the host vs. outputs that are compiled for the target arch above.
	Arch cli.Arch
	// Aggregated coverage for this run
	Coverage TestCoverage
	// True if we want to keep going on build failures and not exit early on the first error encountered
	KeepGoing bool
	// True if we require rule hashes to be correctly verified (usually the case).
	VerifyHashes bool
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
	// Whether and how to download outputs
	OutputDownload OutputDownloadOption
	// True if we only need to parse the initial package (i.e. don't search downwards
	// through deps) - for example when doing `plz query print`.
	ParsePackageOnly bool
	// True if this build is triggered by watching for changes
	Watch bool
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
	// Port specified when debugging a target in server mode.
	DebugPort int
	// True to attach a debugger on test failure.
	DebugFailingTests bool
	// True if we think the underlying filesystem supports xattrs (which affects how we write some metadata).
	XattrsSupported bool
	// Number of times to run each test target. 1 == once each, plus flakes if necessary.
	NumTestRuns uint16
	// Experimental directories
	experimentalLabels []BuildLabel
	// Various items for tracking progress.
	progress *stateProgress
	// CurrentSubrepo is the subrepo this state is for or the empty string if this is the host repo's state
	CurrentSubrepo string
	// ParentState is the state of the repo containing this subrepo. Nil if this is the host repo.
	ParentState *BuildState
	// EnableBreakpoints enablese the breakpoint() build-in, and drops Please into an interactive debugger when
	// they're encountered.
	EnableBreakpoints bool

	// initOnce is used to control loading the subrepo .plzconfig
	initOnce *sync.Once

	// preloadDownloadOnce is used
	preloadDownloadOnce *sync.Once
}

// Copy creates a copy of this state object
func (state *BuildState) Copy() *BuildState {
	ret := &BuildState{}
	*ret = *state

	ret.initOnce = new(sync.Once)
	ret.preloadDownloadOnce = new(sync.Once)
	return ret
}

// Initialise will load the .plzconfig from the subrepo. We can only do this once the subrepo is built hence why
// it's not done up front. Once we have done that, we can initialise the parser for the subrepo.
func (state *BuildState) Initialise(subrepo *Subrepo) (err error) {
	state.initOnce.Do(func() {
		// If we are the root repo, or a cross-compilation of that, we don't want to re-load the config files. That's
		// handled for us already in plz.go
		if state.CurrentSubrepo != "" {
			state.RepoConfig = &Configuration{}
			if err := readSubrepoConfig(state.RepoConfig, subrepo); err != nil {
				return
			}
			if err = validateSubrepoNameAndPluginConfig(state.Config, state.RepoConfig, subrepo); err != nil {
				return
			}
		}
	})
	return
}

// A stateProgress records various points of progress for a State.
// This is split out from above so we can share it between multiple instances.
type stateProgress struct {
	// Used to count the number of currently active/pending targets
	numActive  int64
	numPending int64
	numDone    int64
	mutex      sync.Mutex
	closeOnce  sync.Once
	resultOnce sync.Once
	// Used to track subinclude() calls that block until targets are built. Keyed by their label.
	pendingTargets *cmap.Map[BuildLabel, chan struct{}]
	// Used to track general package parsing requests. Keyed by a packageKey struct.
	pendingPackages *cmap.Map[packageKey, chan struct{}]
	// similar to pendingPackages but consumers haven't committed to parsing the package
	packageWaits *cmap.Map[packageKey, chan struct{}]
	// The set of known states
	allStates []*BuildState
	// Targets that we were originally requested to build
	originalTargets *TargetSet
	// True if something about the build has failed.
	failed atomic.Bool
	// True if >= 1 target has failed to build
	buildFailed atomic.Bool
	// True if >= 1 target has failed test cases
	testFailed atomic.Bool
	// Stream of results from the build
	results chan *BuildResult
	// Internal result stream, used to intermediate them for the cycle checker.
	internalResults chan *BuildResult
	// The cycle checker itself.
	cycleDetector cycleDetector
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

// lockedStats is just SystemStats with a mutex to protect it.
type lockedStats struct {
	sync.Mutex
	Stats SystemStats
}

// addActiveTargets increments the counter for a number of newly active build targets.
func (state *BuildState) addActiveTargets(n int) {
	atomic.AddInt64(&state.progress.numActive, int64(n))
}

// addPendingParse adds a task for a pending parse of a build label.
func (state *BuildState) addPendingParse(label, dependent BuildLabel, mode ParseMode) {
	atomic.AddInt64(&state.progress.numActive, 1)
	atomic.AddInt64(&state.progress.numPending, 1)

	go func() {
		defer func() {
			recover() // Prevent death on 'send on closed channel'
		}()
		state.pendingParses <- ParseTask{Label: label, Dependent: dependent, Mode: mode}
	}()
}

// addPendingBuild adds a task for a pending build of a target.
func (state *BuildState) addPendingBuild(target *BuildTarget) {
	atomic.AddInt64(&state.progress.numPending, 1)
	go func() {
		defer func() {
			recover() // Prevent death on 'send on closed channel'
		}()
		state.pendingActions <- Task{Target: target, Type: BuildTask}
	}()
}

// AddPendingTest adds a task for a pending test of a target.
func (state *BuildState) AddPendingTest(target *BuildTarget) {
	if state.TestSequentially {
		state.addPendingTest(target, 1)
	} else {
		state.addPendingTest(target, int(state.NumTestRuns))
	}
}

func (state *BuildState) addPendingTest(target *BuildTarget, numRuns int) {
	atomic.AddInt64(&state.progress.numPending, int64(numRuns))
	go func() {
		defer func() {
			recover() // Prevent death on 'send on closed channel'
		}()
		for run := 1; run <= numRuns; run++ {
			state.pendingActions <- Task{Target: target, Run: uint32(run), Type: TestTask}
		}
	}()
}

// TaskQueues returns a set of channels to listen on for tasks of various types.
func (state *BuildState) TaskQueues() (parses <-chan ParseTask, actions <-chan Task) {
	return state.pendingParses, state.pendingActions
}

// TaskDone indicates that a single task is finished. Should be called after one is finished with
// a task returned from NextTask().
func (state *BuildState) TaskDone() {
	state.taskDone(false)
}

func (state *BuildState) taskDone(wasSynthetic bool) {
	if !wasSynthetic {
		atomic.AddInt64(&state.progress.numDone, 1)
	}
	if atomic.AddInt64(&state.progress.numPending, -1) <= 0 {
		state.Stop()
	}
}

// Stop stops the worker queues after any current tasks are done.
func (state *BuildState) Stop() {
	state.progress.closeOnce.Do(func() {
		close(state.pendingParses)
		close(state.pendingActions)
	})
}

// CloseResults closes the result channels.
func (state *BuildState) CloseResults() {
	state.progress.cycleDetector.Stop()
	state.progress.mutex.Lock()
	defer state.progress.mutex.Unlock()
	if state.progress.results != nil {
		state.progress.resultOnce.Do(func() {
			close(state.progress.results)
		})
	}
}

// IsOriginalTarget returns true if a target is an original target, ie. one specified on the command line.
func (state *BuildState) IsOriginalTarget(target *BuildTarget) bool {
	return state.isOriginalTarget(target, false)
}

func (state *BuildState) isOriginalTarget(target *BuildTarget, exact bool) bool {
	if exact {
		return state.progress.originalTargets.MatchExact(target.Label)
	}
	matched, wasExact := state.progress.originalTargets.Match(target.Label)
	return matched && (wasExact || state.ShouldInclude(target))
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
	_, arch := SplitSubrepoArch(label.Subrepo)
	if arch != "" {
		state.Graph.AddSubrepo(SubrepoForArch(state, cli.NewArchFromString(arch)))
	}

	// Check it's not excluded first.
	for _, e := range state.ExcludeTargets {
		if e.Includes(label) {
			return
		}
	}
	if addToList {
		state.progress.originalTargets.Add(label)
	}
	state.addPendingParse(label, OriginalTarget, ParseModeNormal)
}

// Hasher returns a PathHasher for the given function (e.g. "SHA1").
func (state *BuildState) Hasher(name string) *fs.PathHasher {
	hasher, present := state.hashers[name]
	if !present {
		log.Fatalf("Unknown hash type %s", name)
	}
	return hasher
}

// OutputHashCheckers returns the subset of hash algos that are appropriate for checking the hashes argument on
// build rules
func (state *BuildState) OutputHashCheckers() []*fs.PathHasher {
	hashCheckers := make([]*fs.PathHasher, 0, len(state.Config.Build.HashCheckers))
	for _, algo := range state.Config.Build.HashCheckers {
		hashCheckers = append(hashCheckers, state.Hasher(algo))
	}
	return hashCheckers
}

// LogParseResult logs the result of a target parsing.
func (state *BuildState) LogParseResult(label BuildLabel, status BuildResultStatus, description string) {
	if status == PackageParsed {
		// We may have parse tasks waiting for this package to exist, check for them.
		key := packageKey{Name: label.PackageName, Subrepo: label.Subrepo}
		if ch := state.progress.pendingPackages.Get(key); ch != nil {
			close(ch) // This signals to anyone waiting that it's done.
		}
		if ch := state.progress.packageWaits.Get(key); ch != nil {
			close(ch) // This signals to anyone waiting that it's done.
		}
		return // We don't notify anything else on these.
	}
	state.logResult(&BuildResult{
		Label:       label,
		Status:      status,
		Err:         nil,
		Description: description,
	})
}

// LogBuildResult logs the result of a target building.
func (state *BuildState) LogBuildResult(target *BuildTarget, status BuildResultStatus, description string) {
	state.logResult(&BuildResult{
		Label:       target.Label,
		target:      target,
		Status:      status,
		Err:         nil,
		Description: description,
	})
	if status == TargetBuilt || status == TargetCached {
		// We may have parse tasks waiting for this guy to build, check for them.
		if ch := state.progress.pendingTargets.Get(target.Label); ch != nil {
			close(ch) // This signals to anyone waiting that it's done.
		}
	}
}

// ArchSubrepoInitialised closes the pending target channel for the non-existent arch subrepo psudo-target
func (state *BuildState) ArchSubrepoInitialised(subrepoLabel BuildLabel) {
	// We may have parse tasks waiting for this guy to build, check for them.
	if ch := state.progress.pendingTargets.Get(subrepoLabel); ch != nil {
		close(ch) // This signals to anyone waiting that it's done.
	}
}

// LogTestRunning logs a target while its tests are running.
func (state *BuildState) LogTestRunning(target *BuildTarget, run int, status BuildResultStatus, message string) {
	// Annotate the message with the run number if appropriate.
	if state.NumTestRuns > 1 {
		message = strings.TrimSuffix(message, "...") + fmt.Sprintf(" (run %d of %d)...", run, state.NumTestRuns)
	}
	state.logResult(&BuildResult{
		Label:       target.Label,
		target:      target,
		Run:         run,
		Status:      status,
		Description: message,
	})
}

// LogTestResult logs the result of a target once its tests have completed.
func (state *BuildState) LogTestResult(target *BuildTarget, run int, status BuildResultStatus, results *TestSuite, coverage *TestCoverage, err error, format string, args ...interface{}) {
	state.logResult(&BuildResult{
		Label:       target.Label,
		target:      target,
		Run:         run,
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
func (state *BuildState) LogBuildError(label BuildLabel, status BuildResultStatus, err error, format string, args ...interface{}) {
	state.logResult(&BuildResult{
		Label:       label,
		Status:      status,
		Err:         err,
		Description: fmt.Sprintf(format, args...),
	})
}

// logResult logs a build result directly to the state's queue.
func (state *BuildState) logResult(result *BuildResult) {
	result.Time = time.Now()
	state.progress.internalResults <- result
	if result.Status.IsFailure() {
		state.progress.failed.Store(true)
		if result.Status == TargetBuildFailed {
			state.progress.buildFailed.Store(true)
		} else if result.Status == TargetTestFailed {
			state.progress.testFailed.Store(true)
		}
	}
}

// forwardResults runs indefinitely, forwarding results from the internal
// channel to the external one. On the way it checks if we need to do
// cycle detection.
func (state *BuildState) forwardResults() {
	defer func() {
		if r := recover(); r != nil {
			// Ensure we don't get a "send on closed channel" when the
			// outward results channel is closed.
			log.Debug("%s", r)
		}
	}()
	activeTargets := map[*BuildTarget]struct{}{}
	// Persist this one timer throughout so we don't generate bazillions of them.
	t := time.NewTimer(cycleCheckDuration)
	t.Stop()
	var result *BuildResult
	for {
		if len(activeTargets) == 0 {
			t.Reset(cycleCheckDuration)
			select {
			case result = <-state.progress.internalResults:
				// This has to be properly managed to prevent hangs.
				if !t.Stop() {
					<-t.C
				}
			case <-t.C:
				go state.checkForCycles()
				// Still need to get a result!
				result = <-state.progress.internalResults
			}
		} else {
			result = <-state.progress.internalResults
		}
		if target := result.target; target != nil {
			if result.Status.IsActive() {
				activeTargets[target] = struct{}{}
			} else {
				delete(activeTargets, target)
			}
		}
		state.progress.mutex.Lock()
		if state.progress.results != nil {
			state.progress.results <- result
		}
		state.progress.mutex.Unlock()
	}
}

// RegisterPreloads waits for all preloaded subinclude targets to be built, downloads them, and then registers them with
// the interpreter. We have to actually register them otherwise this will return before we build any
// transitive subincludes.
func (state *BuildState) RegisterPreloads() error {
	var err error
	state.preloadDownloadOnce.Do(func() {
		var eg errgroup.Group
		for _, inc := range state.GetPreloadedSubincludes() {
			if inc.IsPseudoTarget() {
				log.Fatalf("Can't preload pseudotarget %v", inc)
			}

			// Queue them up asynchronously to feed the queues as quickly as possible
			inc := inc
			eg.Go(func() error {
				state.WaitForTargetAndEnsureDownload(inc, OriginalTarget, true)
				return state.Parser.RegisterPreload(inc)
			})
		}
		// We must wait for all the subinclude targets to be built otherwise updating the locals might race with parsing
		// a package
		err = eg.Wait()
	})
	return err
}

// checkForCycles is run to detect a cycle in the graph. It converts any returned error into an async error.
func (state *BuildState) checkForCycles() {
	if err := state.progress.cycleDetector.Check(); err != nil {
		state.LogBuildError(err.Cycle[0].Label, TargetBuildFailed, err, "")
		state.Stop()
	}
}

// Failures returns anything that has failed about the current build.
func (state *BuildState) Failures() (anything, build, test bool) {
	return state.progress.failed.Load(), state.progress.buildFailed.Load(), state.progress.testFailed.Load()
}

// Results returns a channel on which the caller can listen for results.
func (state *BuildState) Results() <-chan *BuildResult {
	state.progress.mutex.Lock()
	defer state.progress.mutex.Unlock()
	if state.progress.results == nil {
		state.progress.results = make(chan *BuildResult, 1000)
	}
	return state.progress.results
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

// ExpandOriginalLabels expands any pseudo-labels (ie. :all, ... has already been resolved to a bunch :all targets)
// from the set of original labels. This will exclude non-test targets when we're building for test.
func (state *BuildState) ExpandOriginalLabels() BuildLabels {
	return state.ExpandLabels(state.progress.originalTargets.AllTargets())
}

// ExpandAllOriginalLabels is the same as ExpandOriginalLabels except it always includes non-test targets
func (state *BuildState) ExpandAllOriginalLabels() BuildLabels {
	return state.expandLabels(state.progress.originalTargets.AllTargets(), false)
}

func AnnotateLabels(labels []BuildLabel) []AnnotatedOutputLabel {
	ret := make([]AnnotatedOutputLabel, len(labels))
	for i, l := range labels {
		ret[i] = AnnotatedOutputLabel{BuildLabel: l}
	}
	return ret
}

func readingStdinAnnotated(labels []AnnotatedOutputLabel) bool {
	for _, l := range labels {
		if l.BuildLabel == BuildLabelStdin {
			return true
		}
	}
	return false
}

// ExpandOriginalMaybeAnnotatedLabels works the same as ExpandOriginalLabels, however requires that the possitional args
// be passed to it.
func (state *BuildState) ExpandOriginalMaybeAnnotatedLabels(args []AnnotatedOutputLabel) []AnnotatedOutputLabel {
	if readingStdinAnnotated(args) {
		args = AnnotateLabels(state.ExpandOriginalLabels())
	}
	return state.ExpandMaybeAnnotatedLabels(args)
}

// ExpandMaybeAnnotatedLabels is the same as ExpandOriginalLabels except for annotated labels
func (state *BuildState) ExpandMaybeAnnotatedLabels(labels []AnnotatedOutputLabel) []AnnotatedOutputLabel {
	ret := make([]AnnotatedOutputLabel, 0, len(labels))
	for _, l := range labels {
		if l.Annotation != "" {
			ret = append(ret, l)
		} else {
			for _, l := range state.ExpandLabels([]BuildLabel{l.BuildLabel}) {
				ret = append(ret, AnnotatedOutputLabel{BuildLabel: l})
			}
		}
	}

	return ret
}

// ExpandLabels expands any pseudo-labels (ie. :all, ... has already been resolved to a bunch :all targets) from a set of labels.
func (state *BuildState) ExpandLabels(labels []BuildLabel) BuildLabels {
	return state.expandLabels(labels, state.NeedTests)
}

// ExpandLabels expands any pseudo-labels (ie. :all, ... has already been resolved to a bunch :all targets) from a set of labels.
func (state *BuildState) expandLabels(labels []BuildLabel, justTests bool) BuildLabels {
	ret := BuildLabels{}
	for _, label := range labels {
		if label.IsPseudoTarget() {
			ret = append(ret, state.expandOriginalPseudoTarget(label, justTests)...)
		} else {
			ret = append(ret, label)
		}
	}
	return ret
}

// expandOriginalPseudoTarget expands one original pseudo-target (i.e. :all or /...) and sorts it
func (state *BuildState) expandOriginalPseudoTarget(label BuildLabel, justTests bool) BuildLabels {
	ret := BuildLabels{}
	addPackage := func(pkg *Package) {
		for _, target := range pkg.AllTargets() {
			if state.ShouldInclude(target) && (!justTests || target.IsTest()) {
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

// SyncParsePackage either returns the given package which is already parsed and available,
// or returns nil indicating it is ready to be parsed. Everything subsequently calling this
// will block until the original caller parse it.
func (state *BuildState) SyncParsePackage(label BuildLabel) *Package {
	if p := state.Graph.PackageByLabel(label); p != nil {
		return p
	}
	if ch, inserted := state.progress.pendingPackages.AddOrGet(label.packageKey(), make(chan struct{})); !inserted {
		waitOnChan(ch, "Still waiting for SyncParsePackage(%v)", label)
	}
	return state.Graph.PackageByLabel(label) // Important to check again; it's possible to race against this whole lot.
}

func waitOnChan[T any](ch chan T, message string, args ...any) T {
	start := time.Now()
	t := time.NewTimer(10 * time.Second)
	defer t.Stop()
	for {
		select {
		case v := <-ch:
			return v
		case <-t.C:
			log.Debugf("%v (after %v)", fmt.Sprintf(message, args...), time.Since(start))
		}
	}
}

// WaitForPackage is similar to WaitForBuiltTarget however it waits for the package to be parsed, queuing it for parse
// if necessary
func (state *BuildState) WaitForPackage(l, dependent BuildLabel, mode ParseMode) *Package {
	if p := state.Graph.PackageByLabel(l); p != nil {
		return p
	}
	key := packageKey{Name: l.PackageName, Subrepo: l.Subrepo}

	// If something has promised to parse it, wait for them to do so
	if ch := state.progress.pendingPackages.Get(key); ch != nil {
		waitOnChan(ch, "Still waiting for pending package in WaitForPackage(%v, %v, %v)", l, dependent, mode)
		return state.Graph.PackageByLabel(l)
	}

	// If something has already queued the package to be parsed, wait for them
	if ch := state.progress.packageWaits.Get(key); ch != nil {
		waitOnChan(ch, "Still waiting for package wait in WaitForPackage(%v, %v, %v)", l, dependent, mode)
		return state.Graph.PackageByLabel(l)
	}

	// Otherwise queue the target for parse and recurse
	state.addPendingParse(l, dependent, mode)
	state.progress.packageWaits.Set(key, make(chan struct{}))

	return state.WaitForPackage(l, dependent, mode)
}

func (state *BuildState) WaitForBuiltTarget(l, dependent BuildLabel, mode ParseMode) *BuildTarget {
	if t := state.Graph.Target(l); t != nil {
		if t.State().IsBuilt() {
			return t
		}
	}
	dependent.Name = "all" // Every target in this package depends on this one.
	// okay, we need to register and wait for this guy.
	if ch, inserted := state.progress.pendingTargets.AddOrGet(l, make(chan struct{})); !inserted {
		// Something's already registered for this, get on the train
		waitOnChan(ch, "Still waiting on WaitForBuiltTarget(%v, %v, %v)", l, dependent, mode)
		return state.Graph.Target(l)
	}
	if err := state.queueTarget(l, dependent, mode.IsForSubinclude(), mode); err != nil {
		log.Fatalf("%v", err)
	}

	// Do this all over; the re-checking that happens here is actually fairly important to resolve
	// a potential race condition if the target was built between us checking earlier and registering
	// the channel just now.
	return state.WaitForBuiltTarget(l, dependent, mode)
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
		for _, src := range target.AllLocalSourceLocalPaths() {
			pkg.MustRegisterOutput(state, src, target)
		}
	} else {
		for _, out := range target.DeclaredOutputs() {
			pkg.MustRegisterOutput(state, out, target)
		}
		if target.IsTest() {
			for _, out := range target.Test.Outputs {
				if !fs.IsGlob(out) {
					pkg.MustRegisterOutput(state, out, target)
				}
			}
		}
	}
}

// ShouldDownload returns true if the given target should be downloaded during remote execution.
func (state *BuildState) ShouldDownload(target *BuildTarget) bool {
	// Need to download the target if it was originally requested (and the user didn't pass --nodownload).
	downloadOriginalTarget := state.OutputDownload == OriginalOutputDownload && state.IsOriginalTarget(target)
	downloadTransitiveTarget := state.OutputDownload == TransitiveOutputDownload
	downloadLinkableTarget := state.Config.Build.DownloadLinkable && target.HasLinks(state)
	return (downloadOriginalTarget && !state.NeedTests) || downloadTransitiveTarget || downloadLinkableTarget
}

// ShouldRebuild returns true if we should force a rebuild of this target (i.e. the user
// has done plz build --rebuild where we would not otherwise build it).
func (state *BuildState) ShouldRebuild(target *BuildTarget) bool {
	return state.ForceRebuild && state.IsOriginalTargetOrParent(target)
}

// WillRunRemotely returns true if the given target will be run on a remote executor.
func (state *BuildState) WillRunRemotely(target *BuildTarget) bool {
	return state.RemoteClient != nil && state.Config.IsRemoteExecution() && !target.Local
}

// EnsureDownloaded ensures that a target has been downloaded when built remotely.
// If remote execution is not enabled it has no effect.
func (state *BuildState) EnsureDownloaded(target *BuildTarget) error {
	if state.RemoteClient != nil {
		if err := state.RemoteClient.Download(target); err != nil {
			return fmt.Errorf("Failed to download outputs for %s: %s", target, err)
		}
	}
	return nil
}

// WaitForTargetAndEnsureDownload waits for the target to be built and then downloads it if executing remotely
func (state *BuildState) WaitForTargetAndEnsureDownload(l, dependent BuildLabel, isForPreload bool) *BuildTarget {
	mode := ParseModeForSubinclude
	if isForPreload {
		mode |= ParseModeForPreload
	}
	return state.waitForTargetAndEnsureDownload(l, dependent, mode)
}

// WaitForInitialTargetAndEnsureDownload is like WaitForTargetAndEnsureDownload but is used for
// targets in the initial set.
func (state *BuildState) WaitForInitialTargetAndEnsureDownload(l, dependent BuildLabel) *BuildTarget {
	// This may have been an architecture label from the CLI
	if state.WaitForBuiltTarget(l, dependent, ParseModeNormal) == nil {
		return nil
	}
	return state.waitForTargetAndEnsureDownload(l, dependent, ParseModeNormal)
}

func (state *BuildState) waitForTargetAndEnsureDownload(l, dependent BuildLabel, mode ParseMode) *BuildTarget {
	target := state.WaitForBuiltTarget(l, dependent, mode)
	if !target.State().IsBuilt() {
		return nil
	}
	if err := state.EnsureDownloaded(target); err != nil {
		panic(fmt.Errorf("failed to download target outputs: %w", err))
	}
	return target
}

// ActivateTarget marks a target as active (ie. to be built) and adds its dependencies as pending parses.
func (state *BuildState) ActivateTarget(pkg *Package, label, dependent BuildLabel, mode ParseMode) error {
	if !label.IsAllTargets() && state.Graph.Target(label) == nil {
		if label.Subrepo == "" && label.PackageName == "" && label.Name == dependent.Subrepo {
			if subrepo := state.CheckArchSubrepo(label.Name); subrepo != nil {
				state.ArchSubrepoInitialised(label)
				return nil
			}
		}
		if state.Config.Bazel.Compatibility && mode.IsForSubinclude() {
			// Bazel allows some things that look like build targets but aren't - notably the syntax
			// to load(). It suits us to treat that as though it is one, but we now have to
			// implicitly make it available.
			exportFile(state, pkg, label)
		} else {
			msg := fmt.Sprintf("Parsed build file %s but it doesn't contain target %s", pkg.Filename, label.Name)
			if dependent != OriginalTarget {
				msg += fmt.Sprintf(" (depended on by %s)", dependent)
			}
			return fmt.Errorf(msg + suggestTargets(pkg, label, dependent))
		}
	}
	if state.ParsePackageOnly && !mode.IsForSubinclude() {
		return nil // Some kinds of query don't need a full recursive parse.
	} else if label.IsAllTargets() {
		if dependent == OriginalTarget {
			for _, target := range pkg.AllTargets() {
				// Don't activate targets that were added in a post-build function; that causes a race condition
				// between the post-build functions running and other things trying to activate them too early.
				if state.ShouldInclude(target) && !target.AddedPostBuild {
					// Must always do this for coverage because we need to calculate sources of
					// non-test targets later on.
					if !state.NeedTests || target.IsTest() || state.NeedCoverage {
						if err := state.QueueTarget(target.Label, dependent, dependent.IsAllTargets(), mode); err != nil {
							return err
						}
					}
				}
			}
		}
	} else {
		for _, l := range state.Graph.DependentTargets(dependent, label) {
			// We use :all to indicate a dependency needed for parse.
			if err := state.QueueTarget(l, dependent, dependent.IsAllTargets(), mode); err != nil {
				return err
			}
		}
	}
	return nil
}

// exportFile adds a single-file export target. This is primarily used for Bazel compat.
func exportFile(state *BuildState, pkg *Package, label BuildLabel) {
	t := NewBuildTarget(label)
	t.Subrepo = pkg.Subrepo
	t.IsFilegroup = true
	t.AddSource(NewFileLabel(label.Name, pkg))
	state.AddTarget(pkg, t)
}

// CheckArchSubrepo checks if a target refers to a cross-compiling subrepo.
// Those don't have to be explicitly defined - maybe we should insist on that, but it's nicer not to have to.
func (state *BuildState) CheckArchSubrepo(name string) *Subrepo {
	var arch cli.Arch
	if err := arch.UnmarshalFlag(name); err == nil {
		return state.Graph.MaybeAddSubrepo(SubrepoForArch(state, arch))
	}
	return nil
}

// QueueTarget adds a single target to the build queue.
func (state *BuildState) QueueTarget(label, dependent BuildLabel, forceBuild bool, mode ParseMode) error {
	return state.queueTarget(label, dependent, forceBuild || mode.IsForSubinclude() || (mode&ParseModeForceBuild) != 0, mode)
}

func (state *BuildState) queueTarget(label, dependent BuildLabel, forceBuild bool, mode ParseMode) error {
	target := state.Graph.Target(label)
	if target == nil {
		// If the package isn't loaded yet, we need to queue a parse for it.
		if state.Graph.PackageByLabel(label) == nil {
			if forceBuild {
				mode |= ParseModeForceBuild
			}
			// Queue the target up for parse. The parse step activates the target for us if it needs to be built, so we
			// don't need to do this here.
			state.addPendingParse(label, dependent, mode)
			return nil
		}
		// Package is loaded but target doesn't exist in it. Check again to avoid nasty races.
		target = state.Graph.Target(label)
		if target == nil {
			return fmt.Errorf("Target %s (referenced by %s) doesn't exist", label, dependent)
		}
	}
	if dependent.IsAllTargets() || dependent == OriginalTarget {
		return state.queueResolvedTarget(target, forceBuild, mode)
	}
	for _, l := range target.ProvideFor(state.Graph.TargetOrDie(dependent)) {
		if l == label {
			if err := state.queueResolvedTarget(target, forceBuild, mode); err != nil {
				return err
			}
		} else if err := state.queueTarget(l, dependent, forceBuild, mode); err != nil {
			return err
		}
	}
	return nil
}

// QueueTestTarget adds a target to the queue to be tested.
func (state *BuildState) QueueTestTarget(target *BuildTarget) {
	state.queueTargetData(target)
	state.AddPendingTest(target)
}

// queueTargetData queues up builds of the target's runtime data.
func (state *BuildState) queueTargetData(target *BuildTarget) {
	for _, data := range target.AllData() {
		if l, ok := data.Label(); ok {
			state.WaitForBuiltTarget(l, target.Label, ParseModeForSubinclude)
		}
	}
}

// queueResolvedTarget is like queueTarget but once we have a resolved target.
func (state *BuildState) queueResolvedTarget(target *BuildTarget, forceBuild bool, mode ParseMode) error {
	if mode.IsForSubinclude() {
		target.neededForSubinclude.Store(true)
	}
	if target.State() >= Active && !forceBuild {
		return nil // Target is already tagged to be built and likely on the queue.
	}

	queueAsync := func(shouldBuild bool) {
		if target.IsTest() && state.NeedTests {
			if state.TestSequentially {
				state.addActiveTargets(2) // One for build & one for test
			} else {
				// Tests count however many times we're going to run them if parallel.
				state.addActiveTargets(int(1 + state.NumTestRuns))
			}
		} else {
			state.addActiveTargets(1)
		}
		// Actual queuing stuff now happens asynchronously in here.
		atomic.AddInt64(&state.progress.numPending, 1)
		go state.queueTargetAsync(target, forceBuild, shouldBuild, mode)
	}

	// Here we want to ensure we don't queue the target every time; ideally we only do it once.
	// However we might need to do it twice if the initial request doesn't require it to be built
	// but a later one does.
	if state.NeedBuild || forceBuild {
		if target.SyncUpdateState(Inactive, Active) || target.SyncUpdateState(Semiactive, Active) {
			queueAsync(true)
		}
	} else if target.SyncUpdateState(Inactive, Semiactive) {
		queueAsync(false)
	}
	return nil
}

// queueTarget enqueues a target's dependencies and the target itself once they are done.
func (state *BuildState) queueTargetAsync(target *BuildTarget, forceBuild, building bool, mode ParseMode) {
	defer state.taskDone(true)
	for _, dep := range target.DeclaredDependencies() {
		if err := state.queueTarget(dep, target.Label, forceBuild, mode); err != nil {
			state.asyncError(dep, err)
			return
		}
	}
	for {
		var called atomic.Bool
		if err := target.resolveDependencies(state.Graph, func(t *BuildTarget) error {
			called.Store(true)
			return state.queueResolvedTarget(t, forceBuild, ParseModeNormal)
		}); err != nil {
			state.asyncError(target.Label, err)
			return
		}
		// Wait for these targets to actually build.
		if building {
			for _, t := range target.Dependencies() {
				t.WaitForBuild()
				if t.State() >= DependencyFailed { // Either the target failed or its dependencies failed
					// Give up and set the original target as dependency failed
					target.SetState(DependencyFailed)
					state.LogBuildResult(target, TargetBuilt, "Dependency failed")
					target.FinishBuild()
					return
				}
			}
		}
		if !called.Load() {
			// We are now ready to go, we have nothing to wait for.
			if building && target.SyncUpdateState(Active, Pending) {
				// If we're going to run the target, we need its runtime data to be done. This has to
				// happen before we build it otherwise remote downloads will fail.
				if state.NeedRun && state.IsOriginalTarget(target) {
					state.queueTargetData(target)
				}
				state.addPendingBuild(target)
			}
			return
		}
	}
}

// asyncError reports an error that's happened in an asynchronous function.
func (state *BuildState) asyncError(label BuildLabel, err error) {
	log.Error("Error queuing %s: %s", label, err)
	state.LogBuildError(label, TargetBuildFailed, err, "")
	state.Stop()
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
	state.progress.mutex.Lock()
	defer state.progress.mutex.Unlock()

	for _, s := range state.progress.allStates {
		if s.Arch == arch && s.CurrentSubrepo == state.CurrentSubrepo {
			return s
		}
	}

	// Copy with the architecture-specific config file.
	// This is slightly wrong in that other things (e.g. user-specified command line overrides) should
	// in fact take priority over this, but that's a lot more fiddly to get right.

	// Duplicate & alter configuration
	s := state.Copy()

	configPath := ".plzconfig_" + arch.String()

	config := state.Config.copyConfig()
	if err := readConfigFile(fs.HostFS, config, configPath, false); err != nil {
		log.Fatalf("%v", err)
	}

	repoConfig := state.Config.copyConfig()
	if err := readConfigFile(fs.HostFS, repoConfig, configPath, false); err != nil {
		log.Fatalf("%v", err)
	}

	s.Config = config
	s.RepoConfig = repoConfig
	s.Arch = arch
	state.progress.allStates = append(state.progress.allStates, s)

	return s
}

// ForSubrepo creates a new state for the given subrepo
func (state *BuildState) ForSubrepo(name string, bazelCompat bool) *BuildState {
	state.progress.mutex.Lock()
	defer state.progress.mutex.Unlock()

	for _, s := range state.progress.allStates {
		if s.CurrentSubrepo == name {
			return s
		}
	}

	s := state.Copy()

	s.Config = state.Config.copyConfig()

	s.CurrentSubrepo = name
	s.ParentState = state

	if bazelCompat {
		s.Config.Bazel.Compatibility = true
		s.Config.Parse.BuildFileName = append(state.Config.Parse.BuildFileName, "BUILD.bazel")
	}

	state.progress.allStates = append(state.progress.allStates, s)

	return s
}

// GetPreloadedSubincludes gets the preloaded subincludes for this state, de-duplicating if there are duplicates
func (state *BuildState) GetPreloadedSubincludes() []BuildLabel {
	if len(state.RepoConfig.Parse.PreloadSubincludes) == 0 {
		return state.Config.Parse.PreloadSubincludes
	}

	done := map[BuildLabel]struct{}{}
	includes := make([]BuildLabel, 0, len(state.Config.Parse.PreloadSubincludes)+len(state.RepoConfig.Parse.PreloadSubincludes))

	is := append(state.Config.Parse.PreloadSubincludes, state.RepoConfig.Parse.PreloadSubincludes...)

	for _, i := range is {
		if _, ok := done[i]; ok {
			continue
		}

		includes = append(includes, i)
		done[i] = struct{}{}
	}
	return includes
}

// DownloadInputsIfNeeded downloads all the inputs (or runtime files) for a target if we are building remotely.
func (state *BuildState) DownloadInputsIfNeeded(target *BuildTarget, runtime bool) error {
	if state.RemoteClient != nil {
		state.LogBuildResult(target, TargetBuilding, "Downloading inputs...")
		for input := range state.IterInputs(target, runtime) {
			if l, ok := input.Label(); ok {
				dep := state.Graph.TargetOrDie(l)
				if s := dep.State(); s == BuiltRemotely || s == ReusedRemotely {
					if err := state.RemoteClient.Download(dep); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// DownloadAllInputs downloads all inputs (including sources) for the target. Assumes remote execution.
func (state *BuildState) DownloadAllInputs(target *BuildTarget, targetDir string, isTest bool) error {
	return state.RemoteClient.DownloadInputs(target, targetDir, isTest)
}

// IterInputs returns a channel that iterates all the input files needed for a target.
func (state *BuildState) IterInputs(target *BuildTarget, test bool) <-chan BuildInput {
	if !test {
		return IterInputs(state, state.Graph, target, true, target.IsFilegroup)
	}
	ch := make(chan BuildInput)
	go func() {
		ch <- target.Label
		for _, datum := range target.AllData() {
			ch <- datum
		}
		for _, datum := range target.AllTestTools() {
			ch <- datum
		}
		close(ch)
	}()
	return ch
}

// DisableXattrs disables xattr support for this build. This is done for filesystems that
// don't support it.
func (state *BuildState) DisableXattrs() {
	state.XattrsSupported = false
	state.PathHasher.DisableXattrs()
}

func (state *BuildState) Root() *BuildState {
	if state.ParentState == nil {
		return state
	}
	return state.ParentState.Root()
}

func newCRC32() hash.Hash {
	return hash.Hash(crc32.NewIEEE())
}

func newCRC64() hash.Hash {
	return hash.Hash(crc64.New(crc64.MakeTable(crc64.ISO)))
}

func newBlake3() hash.Hash {
	return blake3.New()
}

func newXXHash() hash.Hash {
	return xxhash.New()
}

func executorFromConfig(config *Configuration) *process.Executor {
	tool := config.Sandbox.Tool
	if !filepath.IsAbs(tool) {
		var err error
		tool, err = LookBuildPath(tool, config)
		if err != nil && (config.Sandbox.Build || config.Sandbox.Test) {
			log.Warningf("Can't find sandbox tool %v on the path: %v", config.Sandbox.Tool, err)
		}
	} else if !fs.FileExists(tool) {
		log.Warningf("Sandbox tool doesn't exist: %v", tool)
	}

	return process.NewSandboxingExecutor(
		config.Sandbox.Tool == "" && (config.Sandbox.Build || config.Sandbox.Test),
		process.NamespacingPolicy(config.Sandbox.Namespace),
		tool,
	)
}

// NewBuildState constructs and returns a new BuildState.
// Everyone should use this rather than attempting to construct it themselves;
// callers can't initialise all the required private fields.
func NewBuildState(config *Configuration) *BuildState {
	graph := NewGraph()
	state := &BuildState{
		Graph:          graph,
		pendingParses:  make(chan ParseTask, 10000),
		pendingActions: make(chan Task, 1000),
		hashers: map[string]*fs.PathHasher{
			// For compatibility reasons the sha1 hasher has no suffix.
			"sha1":   fs.NewPathHasher(RepoRoot, config.Build.Xattrs, sha1.New, "sha1"),
			"sha256": fs.NewPathHasher(RepoRoot, config.Build.Xattrs, sha256.New, "sha256"),
			"crc32":  fs.NewPathHasher(RepoRoot, config.Build.Xattrs, newCRC32, "crc32"),
			"crc64":  fs.NewPathHasher(RepoRoot, config.Build.Xattrs, newCRC64, "crc64"),
			"blake3": fs.NewPathHasher(RepoRoot, config.Build.Xattrs, newBlake3, "blake3"),
			"xxhash": fs.NewPathHasher(RepoRoot, config.Build.Xattrs, newXXHash, "xxhash"),
		},
		ProcessExecutor: executorFromConfig(config),
		StartTime:       startTime,
		Config:          config,
		RepoConfig:      config,
		VerifyHashes:    true,
		NeedBuild:       true,
		XattrsSupported: config.Build.Xattrs,
		Coverage:        TestCoverage{Files: map[string][]LineCoverage{}},
		TargetArch:      config.Build.Arch,
		Arch:            cli.HostArch(),
		stats:           &lockedStats{},
		progress: &stateProgress{
			numActive:       1, // One for the initial target adding on the main thread.
			numPending:      1,
			pendingTargets:  cmap.New[BuildLabel, chan struct{}](cmap.DefaultShardCount, hashBuildLabel),
			pendingPackages: cmap.New[packageKey, chan struct{}](cmap.DefaultShardCount, hashPackageKey),
			packageWaits:    cmap.New[packageKey, chan struct{}](cmap.DefaultShardCount, hashPackageKey),
			internalResults: make(chan *BuildResult, 1000),
			cycleDetector:   cycleDetector{graph: graph},
			originalTargets: NewTargetSet(),
		},
		initOnce:            new(sync.Once),
		preloadDownloadOnce: new(sync.Once),
	}

	state.PathHasher = state.Hasher(config.Build.HashFunction)
	state.progress.allStates = []*BuildState{state}
	state.Hashes.Config = config.Hash()
	for _, exp := range config.Parse.ExperimentalDir {
		state.experimentalLabels = append(state.experimentalLabels, BuildLabel{PackageName: exp, Name: "..."})
	}
	go state.forwardResults()
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
	// Timestamp of this event
	Time time.Time
	// Target which has just changed
	Label BuildLabel
	// Target which has changed. Nil if it's a parse action.
	target *BuildTarget
	// Test run index. 0 if not a test.
	Run int
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
	case TargetBuilding, TargetBuildStopped, TargetBuilt, TargetCached, TargetBuildFailed:
		return "Build"
	case TargetTesting, TargetTestStopped, TargetTested, TargetTestFailed:
		return "Test"
	default:
		return "Other"
	}
}

// IsParse returns true if this status is a parse event
func (s BuildResultStatus) IsParse() bool {
	return s == PackageParsing || s == PackageParsed || s == ParseFailed
}

// IsFailure returns true if this status represents a failure.
func (s BuildResultStatus) IsFailure() bool {
	return s == ParseFailed || s == TargetBuildFailed || s == TargetTestFailed
}

// IsActive returns true if this status represents a target that is not yet finished.
func (s BuildResultStatus) IsActive() bool {
	return s == PackageParsing || s == TargetBuilding || s == TargetTesting
}
