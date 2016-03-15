package core

import "bytes"
import "fmt"
import "sort"
import "sync"
import "time"

type BuildLabelPair struct {
	Label    BuildLabel // Label of target to parse
	Dependor BuildLabel // The target that depended on it
}

// Passed about to track the current state of the build.
type BuildState struct {
	Graph *BuildGraph
	// Stream of pending packages to parse
	pendingParses chan *BuildLabelPair
	// Stream of pending targets to build
	pendingBuilds chan BuildLabel
	// Stream of pending tests to run
	pendingTests chan BuildLabel
	// Stream of results from the build
	Results chan *BuildResult
	// Used to signal goroutines to stop once the build is done.
	Stop chan bool
	// Configuration options
	Config Configuration
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

func (state *BuildState) AddPendingParse(label, dependor BuildLabel) {
	state.mutex.Lock()
	state.numActive++
	state.numPending++
	state.mutex.Unlock()
	state.pendingParses <- &BuildLabelPair{label, dependor}
}

func (state *BuildState) AddPendingBuild(label BuildLabel) {
	state.addPending(label, state.pendingBuilds)
}

func (state *BuildState) AddPendingTest(label BuildLabel) {
	if state.NeedTests {
		state.addPending(label, state.pendingTests)
	}
}

// Used to allow receive-only access to these channels.
// Caller should call ProcessedOne *after* they're done handling the result of one of these.
func (state *BuildState) ReceiveChannels() (<-chan *BuildLabelPair, <-chan BuildLabel, <-chan BuildLabel) {
	return state.pendingParses, state.pendingBuilds, state.pendingTests
}

func (state *BuildState) addPending(label BuildLabel, ch chan<- BuildLabel) {
	state.mutex.Lock()
	state.numPending++
	state.mutex.Unlock()
	ch <- label
}

func (state *BuildState) ProcessedOne() {
	state.mutex.Lock()
	state.numDone++
	state.numPending--
	if state.numPending <= 0 {
		state.Stop <- true
	}
	state.mutex.Unlock()
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

func NewBuildState(numThreads int, cache *Cache, verbosity int, config Configuration) *BuildState {
	State = &BuildState{
		Graph: NewGraph(),
		// Buffer the channels, since they will both send & receive on (potentially) the same threads.
		// TODO(pebers): this is rather awkward, they (particularly the parse channel) can block when
		//               given a sufficiently large set of inputs. I'd prefer not to make the buffers
		//               massive since they are dominating quite a bit of our memory usage...
		pendingParses: make(chan *BuildLabelPair, numThreads*100000),
		pendingBuilds: make(chan BuildLabel, numThreads*10000),
		pendingTests:  make(chan BuildLabel, numThreads*10000),
		Results:       make(chan *BuildResult, numThreads*10000),
		Stop:          make(chan bool, numThreads),
		Config:        config,
		Verbosity:     verbosity,
		Cache:         cache,
		VerifyHashes:  true,
		NeedBuild:     true,
		numActive:     1, // One for the initial target adding on the main thread.
		numPending:    1,
		Coverage:      TestCoverage{Files: map[string][]LineCoverage{}},
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
