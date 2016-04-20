package core

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Workiva/go-datastructures/queue"
)

// A TaskType identifies the kind of task returned from NextTask()
type TaskType int

// Note that the ordering of these determines priority.
const (
	Stop TaskType = iota
	// These are for subinclude targets which should be processed before others.
	SubincludeBuild
	SubincludeParse
	Build
	Parse
	Test
)

type pendingTask struct {
	Label    BuildLabel // Label of target to parse
	Dependor BuildLabel // The target that depended on it (only for parse tasks)
	Type     TaskType
}

func (t pendingTask) Compare(that pendingTask) int {
	return t.Type - that.Type
}

// Passed about to track the current state of the build.
type BuildState struct {
	Graph *BuildGraph
	// Stream of pending tasks
	pendingTasks *queue.PriorityQueue
	// Stream of results from the build
	Results chan *BuildResult
	// Used to signal goroutines to stop once the build is done.
	Stop chan bool
	// Configuration options
	Config *Configuration
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
	Cache *Cache
	// Targets that we were originally requested to build
	OriginalTargets []BuildLabel
	// Arguments to tests.
	TestArgs []string
	// Labels of targets that we will include / exclude
	Include, Exclude []string
	// Actual targets to exclude from discovery
	ExcludeTargets []BuildLabel
	// True once the main thread has finished finding / loading targets.
	TargetsLoaded bool
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
	// Number of times to run each test target. 0 == once each, plus flakes if necessary.
	NumTestRuns int
	// True to print the build / test commands as they're run
	PrintCommands bool
	// True to clean working directories after successful builds.
	CleanWorkdirs bool
	// Used to count the number of currently active/pending targets
	numActive  int
	numPending int
	numDone    int
	mutex      sync.Mutex
}

// Singleton instance of one of these. Tried to avoid introducing it but it ended up being
// inevitable to make some of the parsing code work.
var State *BuildState

func (state *BuildState) AddActiveTarget() {
	state.mutex.Lock()
	state.numActive++
	state.mutex.Unlock()
}

func (state *BuildState) AddPendingParse(label, dependor BuildLabel, forSubinclude bool) {
	state.mutex.Lock()
	state.numActive++
	state.numPending++
	state.mutex.Unlock()
	if forSubinclude {
		state.pendingTasks.Push(pendingTask{Label: label, Dependor: dependor, Type: SubincludeParse})
	} else {
		state.pendingTasks.Push(pendingTask{Label: label, Dependor: dependor, Type: Parse})
	}
}

func (state *BuildState) AddPendingBuild(label BuildLabel, forSubinclude bool) {
	if forSubinclude {
		state.addPending(label, SubincludeBuild)
	} else {
		state.addPending(label, Build)
	}
}

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
	task := t.(pendingTask)
	return task.Label, task.Dependor, task.Type
}

func (state *BuildState) addPending(label BuildLabel, t TaskType) {
	state.mutex.Lock()
	state.numPending++
	state.mutex.Unlock()
	state.pendingTasks.Put(pendingTask{Label: label, Type: t})
}

// TaskDone indicates that a single task is finished. Should be called after one is finished with
// a task returned from NextTask().
func (state *BuildState) TaskDone() {
	state.mutex.Lock()
	state.numDone++
	state.numPending--
	if state.numPending <= 0 {
		state.Stop <- true
	}
	state.mutex.Unlock()
}

// Stop adds n stop tasks to the list of pending tasks, which stops n workers.
func (state *BuildState) Stop(n int) {
	for i := 0; i < n; i++ {
		state.pendingTasks.Push(pendingTask{Type: Stop})
	}
}

// IsOriginalTarget returns true if a target is an original target, ie. one specified on the command line.
func (state *BuildState) IsOriginalTarget(label BuildLabel) bool {
	for _, original := range state.OriginalTargets {
		if original == label || (original.IsAllTargets() && original.PackageName == label.PackageName) {
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
			state.ExcludeTargets = append(state.ExcludeTargets, parseMaybeRelativeBuildLabel(e, ""))
		} else {
			state.Exclude = append(state.Exclude, e)
		}
	}
}

// AddOriginalTarget adds one of the original targets and enqueues it for parsing / building.
func (state *BuildState) AddOriginalTarget(label BuildLabel) {
	// Check it's not excluded first.
	for _, e := range state.ExcludeTargets {
		if e.includes(label) {
			return
		}
	}
	state.OriginalTargets = append(state.OriginalTargets, label)
	state.AddPendingParse(label, OriginalTarget, false)
}

func (state *BuildState) LogBuildResult(tid int, label BuildLabel, status BuildResultStatus, description string) {
	state.Results <- &BuildResult{
		ThreadId:    tid,
		Time:        time.Now(),
		Label:       label,
		Status:      status,
		Err:         nil,
		Description: description,
	}
}

func (state *BuildState) LogTestResult(tid int, label BuildLabel, status BuildResultStatus, results TestResults, coverage TestCoverage, err error, format string, args ...interface{}) {
	state.Results <- &BuildResult{
		ThreadId:    tid,
		Time:        time.Now(),
		Label:       label,
		Status:      status,
		Err:         err,
		Description: fmt.Sprintf(format, args...),
		Tests:       results,
	}
	state.mutex.Lock()
	defer state.mutex.Unlock()
	state.Coverage.Aggregate(coverage)
}

func (state *BuildState) LogBuildError(tid int, label BuildLabel, status BuildResultStatus, err error, format string, args ...interface{}) {
	state.Results <- &BuildResult{
		ThreadId:    tid,
		Time:        time.Now(),
		Label:       label,
		Status:      status,
		Err:         err,
		Description: fmt.Sprintf(format, args...),
	}
}

func (state *BuildState) NumActive() int {
	state.mutex.Lock()
	defer state.mutex.Unlock()
	return state.numActive
}

func (state *BuildState) NumDone() int {
	state.mutex.Lock()
	defer state.mutex.Unlock()
	return state.numDone
}

// ExpandOriginalTargets expands any pseudo-targets (ie. :all, ... has already been resolved to a bunch :all targets)
// from the set of original targets.
func (state *BuildState) ExpandOriginalTargets() BuildLabels {
	ret := BuildLabels{}
	for _, label := range state.OriginalTargets {
		if label.IsAllTargets() {
			for _, target := range state.Graph.PackageOrDie(label.PackageName).Targets {
				if target.ShouldInclude(state.Include, state.Exclude) && (!state.NeedTests || target.IsTest) {
					ret = append(ret, target.Label)
				}
			}
		} else {
			ret = append(ret, label)
		}
	}
	sort.Sort(ret)
	return ret
}

func NewBuildState(numThreads int, cache *Cache, verbosity int, config *Configuration) *BuildState {
	State = &BuildState{
		Graph:        NewGraph(),
		pendingTasks: queue.NewPriorityQueue(10000, true), // big hint, why not
		Results:      make(chan *BuildResult, numThreads*10000),
		Config:       config,
		Verbosity:    verbosity,
		Cache:        cache,
		VerifyHashes: true,
		NeedBuild:    true,
		numActive:    1, // One for the initial target adding on the main thread.
		numPending:   1,
		Coverage:     TestCoverage{Files: map[string][]LineCoverage{}},
	}
	State.Hashes.Config = config.Hash()
	State.Hashes.Containerisation = config.ContainerisationHash()
	return State
}

type BuildResult struct {
	// Thread id (or goroutine id, really) that generated this result.
	ThreadId int
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

func NewBuildError(tid int, label BuildLabel, status BuildResultStatus, err error, description string) BuildResult {
	return BuildResult{
		ThreadId:    tid,
		Time:        time.Now(),
		Label:       label,
		Status:      status,
		Err:         err,
		Description: description,
	}
}

type BuildResultStatus int

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

type Cache interface {
	// Stores the results of a single build target.
	Store(target *BuildTarget, key []byte)
	// Stores an extra file against a build target.
	// The file name is relative to the target's out directory.
	StoreExtra(target *BuildTarget, key []byte, file string)
	// Retrieves the results of a single build target.
	// If successful, the outputs will be placed into the output file tree.
	Retrieve(target *BuildTarget, key []byte) bool
	// Retrieves an extra file previously stored by StoreExtra.
	// If successful, the file will be placed into the output file tree.
	RetrieveExtra(target *BuildTarget, key []byte, file string) bool
	// Cleans any artifacts associated with this target from the cache, for any possible key.
	Clean(target *BuildTarget)
}

// This is a pretty simple coverage format; we record one int for each line
// stating what its coverage is.
type TestCoverage struct {
	Tests map[BuildLabel]map[string][]LineCoverage
	Files map[string][]LineCoverage
}

// Aggregates results from that coverage object into this one.
func (this *TestCoverage) Aggregate(that TestCoverage) {
	if this.Tests == nil {
		this.Tests = map[BuildLabel]map[string][]LineCoverage{}
	}
	if this.Files == nil {
		this.Files = map[string][]LineCoverage{}
	}

	// Assume that tests are independent (will currently always be the case).
	for label, coverage := range that.Tests {
		this.Tests[label] = coverage
	}
	// Files are more complex since multiple tests can cover the same file.
	// We take the best result for each line from each test.
	for filename, coverage := range that.Files {
		this.Files[filename] = MergeCoverageLines(this.Files[filename], coverage)
	}
}

func MergeCoverageLines(existing, coverage []LineCoverage) []LineCoverage {
	ret := make([]LineCoverage, len(existing))
	copy(ret, existing)
	for i, line := range coverage {
		if i >= len(ret) {
			ret = append(ret, line)
		} else if coverage[i] > ret[i] {
			ret[i] = coverage[i]
		}
	}
	return ret
}

// Returns an ordered slice of all the files we have coverage information for.
func (this TestCoverage) OrderedFiles() []string {
	files := []string{}
	for file, _ := range this.Files {
		if strings.HasPrefix(file, RepoRoot) {
			file = strings.TrimLeft(file[len(RepoRoot):], "/")
		}
		files = append(files, file)
	}
	sort.Strings(files)
	return files
}

func NewTestCoverage() TestCoverage {
	return TestCoverage{
		Tests: map[BuildLabel]map[string][]LineCoverage{},
		Files: map[string][]LineCoverage{},
	}
}

// Produce a string representation of coverage for serialising to file so we don't
// expose the internal enum values (ordering is important so we may want to insert
// new ones later. This format happens to be the same as the one Phabricator uses,
// which is mildly useful to us since we want to integrate with it anyway. See
// https://secure.phabricator.com/book/phabricator/article/arcanist_coverage/
// for more detail of how it works.
func TestCoverageString(lines []LineCoverage) string {
	var buffer bytes.Buffer
	for _, line := range lines {
		buffer.WriteRune(lineCoverageOutput[line])
	}
	return buffer.String()
}

type LineCoverage uint8

const (
	NotExecutable LineCoverage = iota // Line isn't executable (eg. comment, blank)
	Unreachable   LineCoverage = iota // Line is executable but we've determined it can't be reached. So far not used.
	Uncovered     LineCoverage = iota // Line is executable but isn't covered.
	Covered       LineCoverage = iota // Line is executable and covered.
)

var lineCoverageOutput = [...]rune{'N', 'X', 'U', 'C'} // Corresponds to ordering of enum.
