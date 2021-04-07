package core

import (
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"hash"
	"hash/crc32"
	"hash/crc64"
	"io"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/OneOfOne/cmap"
	"lukechampine.com/blake3"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/process"
)

// startTime is as close as we can conveniently get to process start time.
var startTime = time.Now()

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
	// Streams of pending tasks
	pendingParses chan ParseTask
	pendingBuilds, pendingRemoteBuilds chan BuildTask
	pendingTests, pendingRemoteTests chan TestTask
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
	// The original architecture that the user requested to build for. Either the host arch, --arch, or the arch in the
	// .plzconfig
	TargetArch cli.Arch
	// The architecture this state is for. This might change as we re-parse packages for different architectures e.g.
	// for tools that run on the host vs. outputs that are compiled for the target arch above.
	Arch cli.Arch
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
	// True if we have any remote executors configured.
	anyRemote bool
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
	numDone    int64
	mutex      sync.Mutex
	closeOnce  sync.Once
	// Used to track subinclude() calls that block until targets are built. Keyed by their label.
	pendingTargets *cmap.CMap
	// Used to track general package parsing requests. Keyed by a packageKey struct.
	pendingPackages *cmap.CMap
	// similar to pendingPackages but consumers haven't committed to parsing the package
	packageWaits *cmap.CMap
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

// addActiveTargets increments the counter for a number of newly active build targets.
func (state *BuildState) addActiveTargets(n int) {
	atomic.AddInt64(&state.progress.numActive, int64(n))
}

// addPendingParse adds a task for a pending parse of a build label.
func (state *BuildState) addPendingParse(label, dependent BuildLabel, forSubinclude bool) {
	atomic.AddInt64(&state.progress.numActive, 1)
	atomic.AddInt64(&state.progress.numPending, 1)
	go func() {
		defer func() {
			recover()  // Prevent death on 'send on closed channel'
		}()
		state.pendingParses <- ParseTask{Label: label, Dependent: dependent, ForSubinclude: forSubinclude}
	}()
}

// addPendingBuild adds a task for a pending build of a target.
func (state *BuildState) addPendingBuild(target *BuildTarget) {
	atomic.AddInt64(&state.progress.numPending, 1)
	go func() {
		defer func() {
			recover()  // Prevent death on 'send on closed channel'
		}()
		if state.anyRemote && !target.Local {
			state.pendingRemoteBuilds <- target.Label
		} else {
			state.pendingBuilds <- target.Label
		}
	}()
}

// AddPendingTest adds a task for a pending test of a target.
func (state *BuildState) AddPendingTest(target *BuildTarget) {
	if state.TestSequentially {
		state.addPendingTest(target, 1)
	} else {
		state.addPendingTest(target, state.NumTestRuns)
	}
}

func (state *BuildState) addPendingTest(target *BuildTarget, numRuns int) {
	atomic.AddInt64(&state.progress.numPending, int64(numRuns))
	go func() {
		defer func() {
			recover()  // Prevent death on 'send on closed channel'
		}()
		ch := state.pendingTests
		if state.anyRemote && !target.Local {
			ch = state.pendingRemoteTests
		}
		for run := 1; run <= numRuns; run++ {
			ch <- TestTask{Label: target.Label, Run: run}
		}
	}()
}

// TaskQueues returns a set of channels to listen on for tasks of various types.
func (state *BuildState) TaskQueues() (parses <-chan ParseTask, builds, remoteBuilds <-chan BuildTask, tests, remoteTests <-chan TestTask) {
	return state.pendingParses, state.pendingBuilds, state.pendingRemoteBuilds, state.pendingTests, state.pendingRemoteTests
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
	close(state.pendingParses)
	close(state.pendingBuilds)
	close(state.pendingRemoteBuilds)
	close(state.pendingTests)
	close(state.pendingRemoteTests)
	state.CloseResults()
}

// CloseResults closes the result channels.
func (state *BuildState) CloseResults() {
	if state.results != nil {
		state.progress.closeOnce.Do(func() {
			close(state.results)
		})
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
	state.addPendingParse(label, OriginalTarget, false)
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
	return []*fs.PathHasher{state.Hasher("sha1"), state.Hasher("sha256"), state.Hasher("blake3")}
}

// LogBuildResult logs the result of a target either building or parsing.
func (state *BuildState) LogBuildResult(tid int, label BuildLabel, status BuildResultStatus, description string) {
	if status == PackageParsed {
		// We may have parse tasks waiting for this package to exist, check for them.
		if ch, present := state.progress.pendingPackages.GetOK(packageKey{Name: label.PackageName, Subrepo: label.Subrepo}); present {
			close(ch.(chan struct{})) // This signals to anyone waiting that it's done.
		}
		if ch, present := state.progress.packageWaits.GetOK(packageKey{Name: label.PackageName, Subrepo: label.Subrepo}); present {
			close(ch.(chan struct{})) // This signals to anyone waiting that it's done.
		}
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
		if ch, present := state.progress.pendingTargets.GetOK(label); present {
			close(ch.(chan struct{})) // This signals to anyone waiting that it's done.
		}
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

// ExpandOriginalLabels expands any pseudo-labels (ie. :all, ... has already been resolved to a bunch :all targets)
// from the set of original labels.
func (state *BuildState) ExpandOriginalLabels() BuildLabels {
	state.progress.originalTargetMutex.Lock()
	targets := state.progress.originalTargets[:]
	state.progress.originalTargetMutex.Unlock()
	return state.ExpandLabels(targets)
}

func annotateLabels(labels []BuildLabel) []AnnotatedOutputLabel {
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
		args = annotateLabels(state.ExpandOriginalLabels())
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

// SyncParsePackage either returns the given package which is already parsed and available,
// or returns nil indicating it is ready to be parsed. Everything subsequently calling this
// will block until the original caller parse it.
func (state *BuildState) SyncParsePackage(label BuildLabel) *Package {
	if p := state.Graph.PackageByLabel(label); p != nil {
		return p
	}
	var ch chan struct{}
	state.progress.pendingPackages.Update(label.packageKey(), func(old interface{}) interface{} {
		if old != nil {
			ch = old.(chan struct{})
			return old
		}
		return make(chan struct{})
	})
	if ch != nil {
		<-ch
	}
	return state.Graph.PackageByLabel(label) // Important to check again; it's possible to race against this whole lot.
}

// WaitForPackage is similar to WaitForBuiltTarget however
func (state *BuildState) WaitForPackage(l, dependent BuildLabel) *Package {
	if p := state.Graph.PackageByLabel(l); p != nil {
		return p
	}
	key := packageKey{Name: l.PackageName, Subrepo: l.Subrepo}

	// If something has promised to parse it, wait for them to do so
	if ch, present := state.progress.pendingPackages.GetOK(key); present {
		<-ch.(chan struct{})
		return state.Graph.PackageByLabel(l)
	}

	// If something has already queued the package to be parsed, wait for them
	if ch, present := state.progress.packageWaits.GetOK(key); present {
		<-ch.(chan struct{})
		return state.Graph.PackageByLabel(l)
	}

	// Otherwise queue the target for parse and recurse
	state.addPendingParse(l, dependent, true)
	state.progress.packageWaits.Set(key, make(chan struct{}))

	return state.WaitForPackage(l, dependent)
}

// WaitForBuiltTarget blocks until the given label is available as a build target and has been successfully built.
func (state *BuildState) WaitForBuiltTarget(l, dependent BuildLabel) *BuildTarget {
	if t := state.Graph.Target(l); t != nil {
		if s := t.State(); s >= Built && s != Failed {
			// Ensure we have downloaded its outputs if needed.
			// This is a bit fiddly but works around the case where we already built it but
			// didn't download, and now have found we need to.
			state.mustEnsureDownloaded(t)
			return t
		}
	}
	dependent.Name = "all" // Every target in this package depends on this one.
	// okay, we need to register and wait for this guy.
	var ch chan struct{}
	state.progress.pendingTargets.Update(l, func(old interface{}) interface{} {
		if old != nil {
			ch = old.(chan struct{})
			return old
		}
		return make(chan struct{})
	})
	if ch != nil {
		// Something's already registered for this, get on the train
		<-ch
		t := state.Graph.Target(l)
		state.mustEnsureDownloaded(t)
		return t
	}
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
	return state.IsOriginalTarget(target) && state.DownloadOutputs && !state.NeedTests
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

// mustEnsureDownloaded is like EnsureDownloaded but panics on error.
func (state *BuildState) mustEnsureDownloaded(target *BuildTarget) {
	if err := state.EnsureDownloaded(target); err != nil {
		panic(err)
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
			state.addPendingParse(label, dependent, forceBuild)
			return nil
		}
		// Package is loaded but target doesn't exist in it. Check again to avoid nasty races.
		target = state.Graph.Target(label)
		if target == nil {
			return fmt.Errorf("Target %s (referenced by %s) doesn't exist", label, dependent)
		}
	}
	return state.queueResolvedTarget(target, dependent, rescan, forceBuild, neededForSubinclude)
}

// queueResolvedTarget is like queueTarget but once we have a resolved target.
func (state *BuildState) queueResolvedTarget(target *BuildTarget, dependent BuildLabel, rescan, forceBuild, neededForSubinclude bool) error {
	if target.State() >= Active && !rescan && !forceBuild {
		return nil // Target is already tagged to be built and likely on the queue.
	}
	target.NeededForSubinclude = target.NeededForSubinclude || neededForSubinclude

	queueAsync := func(shouldBuild bool) {
		if target.IsTest && state.NeedTests {
			if state.TestSequentially {
				state.addActiveTargets(2) // One for build & one for test
			} else {
				// Tests count however many times we're going to run them if parallel.
				state.addActiveTargets(1 + state.NumTestRuns)
			}
		} else {
			state.addActiveTargets(1)
		}
		// Actual queuing stuff now happens asynchronously in here.
		atomic.AddInt64(&state.progress.numPending, 1)
		go state.queueTargetAsync(target, dependent, rescan, forceBuild, neededForSubinclude, shouldBuild)
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
func (state *BuildState) queueTargetAsync(target *BuildTarget, dependent BuildLabel, rescan, forceBuild, forSubinclude, building bool) {
	defer state.taskDone(true)
	for _, dep := range target.DeclaredDependencies() {
		if err := state.queueTarget(dep, target.Label, rescan, forceBuild, forSubinclude); err != nil {
			state.asyncError(dep, err)
			return
		}
	}
	for {
		called := false
		if err := target.resolveDependencies(state.Graph, func(t *BuildTarget) error {
			called = true
			state.Graph.cycleDetector.AddDependency(target.Label, t.Label)
			return state.queueResolvedTarget(t, target.Label, rescan, forceBuild, forSubinclude)
		}); err != nil {
			state.asyncError(target.Label, err)
			return
		}
		// Wait for these targets to actually build.
		if building {
			for _, t := range target.Dependencies() {
				t.WaitForBuild()
			}
		}
		if !called {
			// We are now ready to go, we have nothing to wait for.
			if building && target.SyncUpdateState(Active, Pending) {
				state.addPendingBuild(target)
			}
			return
		}
	}
}

// asyncError reports an error that's happened in an asynchronous function.
func (state *BuildState) asyncError(label BuildLabel, err error) {
	log.Error("Error queuing %s: %s", label, err)
	state.LogBuildError(0, label, TargetBuildFailed, err, "")
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
	// Check if we've got this one already.
	// N.B. This implicitly handles the case of the host architecture
	if s := state.findArch(arch); s != nil {
		return s
	}
	// Copy with the architecture-specific config file.
	// This is slightly wrong in that other things (e.g. user-specified command line overrides) should
	// in fact take priority over this, but that's a lot more fiddly to get right.
	s := state.ForConfig(".plzconfig_" + arch.String())
	s.Arch = arch
	return s
}

// findArch returns an existing state for the given architecture, if one exists.
func (state *BuildState) findArch(arch cli.Arch) *BuildState {
	state.progress.mutex.Lock()
	defer state.progress.mutex.Unlock()
	for _, s := range state.progress.allStates {
		if s.Arch == arch {
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

// DownloadInputsIfNeeded downloads all the inputs (or runtime files) for a target if we are building remotely.
func (state *BuildState) DownloadInputsIfNeeded(tid int, target *BuildTarget, runtime bool) error {
	if state.RemoteClient != nil {
		state.LogBuildResult(tid, target.Label, TargetBuilding, "Downloading inputs...")
		for input := range state.IterInputs(target, runtime) {
			if l := input.Label(); l != nil {
				dep := state.Graph.TargetOrDie(*l)
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

// IterInputs returns a channel that iterates all the input files needed for a target.
func (state *BuildState) IterInputs(target *BuildTarget, test bool) <-chan BuildInput {
	if !test {
		return IterInputs(state.Graph, target, true, target.IsFilegroup)
	}
	ch := make(chan BuildInput)
	go func() {
		ch <- target.Label
		for _, datum := range target.AllData() {
			ch <- datum
		}
		for _, datum := range target.TestTools() {
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

func newCRC32() hash.Hash {
	return hash.Hash(crc32.NewIEEE())
}

func newCRC64() hash.Hash {
	return hash.Hash(crc64.New(crc64.MakeTable(crc64.ISO)))
}

func newBlake3() hash.Hash {
	// 32 bytes == 256 bits
	return blake3.New(32, nil)
}

// NewBuildState constructs and returns a new BuildState.
// Everyone should use this rather than attempting to construct it themselves;
// callers can't initialise all the required private fields.
func NewBuildState(config *Configuration) *BuildState {
	// Deliberately ignore the error here so we don't require the sandbox tool until it's needed.
	sandboxTool, _ := LookBuildPath(config.Sandbox.Tool, config)
	state := &BuildState{
		Graph:         NewGraph(),
		pendingParses: make(chan ParseTask, 10000),
		pendingBuilds: make(chan BuildTask, 1000),
		pendingRemoteBuilds: make(chan BuildTask, 1000),
		pendingTests: make(chan TestTask, 1000),
		pendingRemoteTests: make(chan TestTask, 1000),
		hashers: map[string]*fs.PathHasher{
			// For compatibility reasons the sha1 hasher has no suffix.
			"sha1":   fs.NewPathHasher(RepoRoot, config.Build.Xattrs, sha1.New, "sha1"),
			"sha256": fs.NewPathHasher(RepoRoot, config.Build.Xattrs, sha256.New, "sha256"),
			"crc32":  fs.NewPathHasher(RepoRoot, config.Build.Xattrs, newCRC32, "crc32"),
			"crc64":  fs.NewPathHasher(RepoRoot, config.Build.Xattrs, newCRC64, "crc64"),
			"blake3": fs.NewPathHasher(RepoRoot, config.Build.Xattrs, newBlake3, "blake3"),
		},
		ProcessExecutor: process.New(sandboxTool),
		StartTime:       startTime,
		Config:          config,
		VerifyHashes:    true,
		NeedBuild:       true,
		XattrsSupported: config.Build.Xattrs,
		anyRemote:       config.NumRemoteExecutors() > 0,
		Coverage:        TestCoverage{Files: map[string][]LineCoverage{}},
		TargetArch:      config.Build.Arch,
		Arch:            cli.HostArch(),
		Stats:           &SystemStats{},
		progress: &stateProgress{
			numActive:       1, // One for the initial target adding on the main thread.
			numPending:      1,
			pendingPackages: cmap.New(),
			pendingTargets:  cmap.New(),
			packageWaits:    cmap.New(),
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
