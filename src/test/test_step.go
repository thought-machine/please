package test

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/build"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/worker"
)

var log = logging.MustGetLogger("test")

const dummyOutput = "=== RUN DummyTest\n--- PASS: DummyTest (0.00s)\nPASS\n"
const dummyCoverage = "<?xml version=\"1.0\" ?><coverage></coverage>"

// Tag that we attach for xattrs to store hashes against files.
// Note that we are required to provide the user namespace; that seems to be set implicitly
// by the attr utility, but that is not done for us here.
const xattrName = "user.plz_test"

var numUploadFailures int64

const maxUploadFailures int64 = 10

// Test runs the tests for a single target.
func Test(tid int, state *core.BuildState, label core.BuildLabel, remote bool, run int) {
	target := state.Graph.TargetOrDie(label)

	// Defer this so that no matter what happens in this test run, we always call target.CompleteRun
	defer func() {
		runsAllCompleted := target.CompleteRun(state)
		if runsAllCompleted && state.Config.Test.Upload != "" {
			if numUploadFailures < maxUploadFailures {
				if err := uploadResults(target, state.Config.Test.Upload.String(), state.Config.Test.UploadGzipped, state.Config.Test.StoreTestOutputOnSuccess); err != nil {
					log.Warning("%s", err)
					if atomic.AddInt64(&numUploadFailures, 1) >= maxUploadFailures {
						log.Error("Failed to upload test results %d times, giving up", maxUploadFailures)
					}
				}
			}
		}
	}()

	state.LogBuildResult(tid, label, core.TargetTesting, "Testing...")
	test(tid, state.ForTarget(target), label, target, remote, run)
}

func test(tid int, state *core.BuildState, label core.BuildLabel, target *core.BuildTarget, runRemotely bool, run int) {
	hash, err := runtimeHash(tid, state, target, runRemotely, run)
	if err != nil {
		state.LogBuildError(tid, label, core.TargetTestFailed, err, "Failed to calculate target hash")
		return
	}

	outputFile := path.Join(target.TestDir(run), core.TestResultsFile)
	coverageFile := path.Join(target.TestDir(run), core.CoverageFile)
	needCoverage := target.NeedCoverage(state)

	// If the user passed --shell then just prepare the directory.
	if state.PrepareShell {
		if err := state.DownloadInputsIfNeeded(tid, target, true); err != nil {
			state.LogBuildError(tid, label, core.TargetTestFailed, err, "Failed to download test inputs")
			return
		}
		if err := prepareTestDir(state, target, run); err != nil {
			state.LogBuildError(tid, label, core.TargetTestFailed, err, "Failed to prepare test directory")
			return
		}
		target.SetState(core.Stopped)
		state.LogBuildResult(tid, label, core.TargetTestStopped, "Test stopped")
		return
	}

	cachedTestResults := func() *core.TestSuite {
		log.Debug("Not re-running test %s; got cached results.", label)
		coverage := parseCoverageFile(target, target.CoverageFile(), run)
		results, err := parseTestResultsFile(target.TestResultsFile())
		results.Package = strings.ReplaceAll(target.Label.PackageName, "/", ".")
		results.Name = target.Label.Name
		results.Cached = true
		if err != nil {
			log.Warningf("Failed to parse cached test file (for %v), Rerunning test. %w", target.Label, err)
			state.Cache.Clean(target)
			return nil
		} else if !results.TestCases.AllSucceeded() {
			log.Warning("Test results (for %s) with failures shouldn't be cached - ignoring.", label)
			state.Cache.Clean(target)
			return nil
		} else {
			logTestSuccess(state, tid, label, &results, coverage)
		}
		return &results
	}

	moveAndCacheOutputFiles := func(results *core.TestSuite, coverage *core.TestCoverage) bool {
		// Never cache test results when given arguments; the results may be incomplete.
		if len(state.TestArgs) > 0 {
			log.Debug("Not caching results for %s, we passed it arguments", label)
			return true
		}
		// Never cache test results if there were failures (usually flaky tests).
		if results.Failures() > 0 {
			log.Debug("Not caching results for %s, test had failures", label)
			return true
		}
		outs := []string{path.Base(target.TestResultsFile())}
		if err := moveOutputFile(state, hash, outputFile, target.TestResultsFile(), dummyOutput); err != nil {
			state.LogTestResult(tid, label, core.TargetTestFailed, results, coverage, err, "Failed to move test output file")
			return false
		}

		if needCoverage || core.PathExists(coverageFile) {
			if err := moveOutputFile(state, hash, coverageFile, target.CoverageFile(), dummyCoverage); err != nil {
				state.LogTestResult(tid, label, core.TargetTestFailed, results, coverage, err, "Failed to move test coverage file")
				return false
			}
			outs = append(outs, path.Base(target.CoverageFile()))
		}
		for _, output := range target.TestOutputs {
			tmpFile := path.Join(target.TestDir(run), output)
			outFile := path.Join(target.OutDir(), output)
			if err := moveOutputFile(state, hash, tmpFile, outFile, ""); err != nil {
				state.LogTestResult(tid, label, core.TargetTestFailed, results, coverage, err, "Failed to move test output file")
				return false
			}
			outs = append(outs, output)
		}
		if state.Cache != nil && !runRemotely {
			state.Cache.Store(target, hash, outs)
		}
		return true
	}

	needToRun := func() bool {
		if state.ForceRerun {
			return true
		}

		if s := target.State(); (s == core.Unchanged || s == core.Reused) && core.PathExists(target.TestResultsFile()) {
			// Output file exists already and appears to be valid. We might still need to rerun though
			// if the coverage files aren't available.
			if needCoverage && !verifyHash(state, target.CoverageFile(), hash) {
				log.Debug("Rerunning %s, coverage file doesn't exist or has wrong hash", target.Label)
				return true
			} else if !verifyHash(state, target.TestResultsFile(), hash) {
				log.Debug("Rerunning %s, results file has incorrect hash", target.Label)
				return true
			}
			return false
		}
		log.Debug("Output file %s does not exist for %s", target.TestResultsFile(), target.Label)
		// Check the cache for these artifacts.
		files := []string{path.Base(target.TestResultsFile())}
		if needCoverage {
			files = append(files, path.Base(target.CoverageFile()))
		}
		return !retrieveFromCache(state, target, hash, files)
	}

	// Don't cache when doing multiple runs, presumably the user explicitly wants to check it.
	if state.NumTestRuns == 1 && !runRemotely && !needToRun() {
		if cachedResults := cachedTestResults(); cachedResults != nil {
			target.Results = *cachedResults
			return
		}
	}

	// Remove any cached test result file.
	if err := RemoveTestOutputs(target); err != nil {
		state.LogBuildError(tid, label, core.TargetTestFailed, err, "Failed to remove test output files")
		return
	}
	if err := startTestWorkerIfNeeded(tid, state, target); err != nil {
		state.LogBuildError(tid, label, core.TargetTestFailed, fmt.Errorf("failed to start test worker: %w", err), "Failed to start test worker")
		return
	}

	target.StartTestSuite()

	coverage := &core.TestCoverage{}
	if state.NumTestRuns == 1 {
		var results core.TestSuite
		results, coverage = doFlakeRun(tid, state, target, runRemotely)
		target.AddTestResults(results)

		if target.Results.TestCases.AllSucceeded() {
			// Success, store in cache
			moveAndCacheOutputFiles(&target.Results, coverage)
		}
	} else if state.TestSequentially {
		for run := 1; run <= state.NumTestRuns; run++ {
			state.LogBuildResult(tid, target.Label, core.TargetTesting, getRunStatus(run, state.NumTestRuns))
			var results core.TestSuite
			results, coverage = doTest(tid, state, target, runRemotely, 1) // Sequential tests re-use run 1's test dir
			target.AddTestResults(results)
		}
	} else {
		state.LogBuildResult(tid, target.Label, core.TargetTesting, getRunStatus(run, state.NumTestRuns))
		var results core.TestSuite
		results, coverage = doTest(tid, state, target, runRemotely, run)
		target.AddTestResults(results)
	}

	logTargetResults(tid, state, target, coverage, run)
}

func retrieveFromCache(state *core.BuildState, target *core.BuildTarget, hash []byte, files []string) bool {
	if state.Cache == nil {
		return false
	}
	if state.Cache.Retrieve(target, hash, files) {
		// Record the xattr if we might've retrieved from the http cache.
		if state.Config.Cache.HTTPURL != "" {
			for _, f := range files {
				fullPath := filepath.Join(target.OutDir(), f)
				if err := fs.RecordAttr(fullPath, hash, xattrName, state.XattrsSupported); err != nil {
					log.Warningf("%v failed to set hash on %s: %w", target, fullPath, err)
					return false
				}
			}
		}
		return true // got from cache
	}
	return false
}

// doFlakeRun runs a test repeatably until it succeeds or exceeds the max number of flakes for the test
func doFlakeRun(tid int, state *core.BuildState, target *core.BuildTarget, runRemotely bool) (core.TestSuite, *core.TestCoverage) {
	coverage := &core.TestCoverage{}
	results := core.TestSuite{}

	// New group of test cases for each group of flaky runs
	for flakes := 1; flakes <= target.Flakiness; flakes++ {
		state.LogBuildResult(tid, target.Label, core.TargetTesting, getFlakeStatus(flakes, target.Flakiness))

		testSuite, cov := doTest(tid, state, target, runRemotely, 1) // If we're running flakes, numRuns must be 1

		results.TimedOut = results.TimedOut || testSuite.TimedOut
		results.Properties = testSuite.Properties
		results.Duration += testSuite.Duration
		// Each set of executions is treated as a group
		// So if a test flakes three times, three executions will be part of one test case.
		results.Add(testSuite.TestCases...)
		coverage.Aggregate(cov)

		// If execution succeeded, we can break out of the flake loop
		if testSuite.TestCases.AllSucceeded() {
			results.Cached = testSuite.Cached
			break
		}
	}

	return results, coverage
}

func getFlakeStatus(flake int, flakes int) string {
	if flakes == 1 {
		return "Testing..."
	}
	return fmt.Sprintf("Testing (flake %d of %d)...", flake, flakes)
}

func getRunStatus(run int, numRuns int) string {
	if numRuns == 1 {
		return "Testing..."
	}
	return fmt.Sprintf("Testing (run %d of %d)...", run, numRuns)
}

func logTargetResults(tid int, state *core.BuildState, target *core.BuildTarget, coverage *core.TestCoverage, run int) {
	if target.Results.TestCases.AllSucceeded() {
		// Clean up the test directory.
		if state.CleanWorkdirs {
			if err := os.RemoveAll(target.TestDir(run)); err != nil {
				log.Warning("Failed to remove test directory for %s: %s", target.Label, err)
			}
		}
		logTestSuccess(state, tid, target.Label, &target.Results, coverage)
		return
	}
	var resultErr error
	var resultMsg string
	if target.Results.Failures() > 0 {
		resultMsg = "Tests failed"
		for _, testCase := range target.Results.TestCases {
			if len(testCase.Failures()) > 0 {
				resultErr = fmt.Errorf(testCase.Failures()[0].Failure.Message)
			}
		}
	} else if target.Results.Errors() > 0 {
		resultMsg = "Tests errored"
		for _, testCase := range target.Results.TestCases {
			if len(testCase.Errors()) > 0 {
				resultErr = fmt.Errorf(testCase.Errors()[0].Error.Message)
			}
		}
	} else {
		resultErr = fmt.Errorf("unknown error")
		resultMsg = "Something went wrong"
	}
	state.LogTestResult(tid, target.Label, core.TargetTestFailed, &target.Results, coverage, resultErr, resultMsg)
}

func logTestSuccess(state *core.BuildState, tid int, label core.BuildLabel, results *core.TestSuite, coverage *core.TestCoverage) {
	var description string
	tests := pluralise("test", results.Tests())
	if results.Skips() != 0 {
		description = fmt.Sprintf("%d %s passed. %d skipped",
			results.Tests(), tests, results.Skips())
	} else {
		description = fmt.Sprintf("%d %s passed.", len(results.TestCases), tests)
	}
	state.LogTestResult(tid, label, core.TargetTested, results, coverage, nil, description)
}

func pluralise(word string, quantity int) string {
	if quantity == 1 {
		return word
	}
	return word + "s"
}

func prepareTestDir(state *core.BuildState, target *core.BuildTarget, run int) error {
	if err := os.RemoveAll(target.TestDir(run)); err != nil {
		return err
	}
	if err := os.MkdirAll(target.TestDir(run), core.DirPermissions); err != nil {
		return err
	}
	if err := state.EnsureDownloaded(target); err != nil {
		return err
	}
	for out := range core.IterRuntimeFiles(state.Graph, target, true, target.TestDir(run)) {
		if err := core.PrepareSourcePair(out); err != nil {
			return err
		}
	}
	return nil
}

// testCommandAndEnv returns the test command & environment for a target.
func testCommandAndEnv(state *core.BuildState, target *core.BuildTarget, run int) (string, []string, error) {
	replacedCmd, err := core.ReplaceTestSequences(state, target, target.GetTestCommand(state))
	env := core.TestEnvironment(state, target, path.Join(core.RepoRoot, target.TestDir(run)))
	if len(state.TestArgs) > 0 {
		args := strings.Join(state.TestArgs, " ")
		replacedCmd += " " + args
		env = append(env, "TESTS="+args)
	}
	return replacedCmd, env, err
}

func runTest(state *core.BuildState, target *core.BuildTarget, run int) ([]byte, error) {
	replacedCmd, env, err := testCommandAndEnv(state, target, run)
	if err != nil {
		return nil, err
	}
	log.Debugf("Running test %s#%d\nENVIRONMENT:\n%s\n%s", target.Label, run, strings.Join(env, "\n"), replacedCmd)
	_, stderr, err := state.ProcessExecutor.ExecWithTimeoutShellStdStreams(target, target.TestDir(run), env, target.TestTimeout, state.ShowAllOutput, replacedCmd, target.TestSandbox, state.DebugTests)
	return stderr, err
}

func doTest(tid int, state *core.BuildState, target *core.BuildTarget, runRemotely bool, run int) (core.TestSuite, *core.TestCoverage) {
	startTime := time.Now()
	metadata, resultsData, coverage, err := doTestResults(tid, state, target, runRemotely, run)
	duration := time.Since(startTime)
	parsedSuite := parseTestOutput(string(metadata.Stdout), string(metadata.Stderr), err, duration, target, resultsData)
	return core.TestSuite{
		Package:    strings.ReplaceAll(target.Label.PackageName, "/", "."),
		Name:       target.Label.Name,
		Duration:   duration,
		TimedOut:   err == context.DeadlineExceeded,
		Properties: parsedSuite.Properties,
		TestCases:  parsedSuite.TestCases,
		Cached:     metadata.Cached,
	}, coverage
}

func doTestResults(tid int, state *core.BuildState, target *core.BuildTarget, runRemotely bool, run int) (*core.BuildMetadata, [][]byte, *core.TestCoverage, error) {
	var err error
	var metadata *core.BuildMetadata

	if runRemotely {
		metadata, err = state.RemoteClient.Test(tid, target, run)
		if metadata == nil {
			metadata = new(core.BuildMetadata)
		}
	} else {
		var stdout []byte
		stdout, err = prepareAndRunTest(tid, state, target, run)
		metadata = &core.BuildMetadata{Stdout: stdout}
	}

	coverage := parseCoverageFile(target, path.Join(target.TestDir(run), core.CoverageFile), run)

	var data [][]byte
	// If this test is meant to produce an output file and the test ran successfully
	if !target.NoTestOutput {
		d, readErr := readTestResultsDir(path.Join(target.TestDir(run), core.TestResultsFile))
		if readErr != nil {
			// If we got an error running the tests, this is probably to be expected and not worth warning about
			if err == nil {
				log.Warningf("failed to read test results file: %v", readErr)
			}
		} else {
			data = d
		}
	}
	return metadata, data, coverage, err
}

// prepareAndRunTest sets up a test directory and runs the test.
func prepareAndRunTest(tid int, state *core.BuildState, target *core.BuildTarget, run int) (stdout []byte, err error) {
	if err = prepareTestDir(state, target, run); err != nil {
		state.LogBuildError(tid, target.Label, core.TargetTestFailed, err, "Failed to prepare test directory for %s: %s", target.Label, err)
		return []byte{}, err
	}
	return runTest(state, target, run)
}

func parseTestOutput(stdout string, stderr string, runError error, duration time.Duration, target *core.BuildTarget, resultsData [][]byte) core.TestSuite {
	// This is all pretty involved; there are lots of different possibilities of what could happen.
	// The contract is that the test must return zero on success or non-zero on failure (Unix FTW).
	// If it's successful, it must produce a parseable file named "test.results" in its temp folder.
	// (alternatively, this can be a directory containing parseable files).
	// Tests can opt out of the file requirement individually, in which case they're judged only
	// by their return value.
	// But of course, we still have to consider all the alternatives here and handle them nicely.

	// No output and no execution error and output not expected - OK
	// No output and no execution error and output expected - SYNTHETIC ERROR - Missing Results
	// No output and execution error - SYNTHETIC ERROR - Failed to Run
	// Output fails to parse - SYNTHETIC ERROR - Failed to parse output
	// Output fails to parse with execution error - SYNTHETIC ERROR + EXECUTION ERROR - Failed to parse output
	// Output and no execution error - PARSE OUTPUT - Ignore noTestOutput
	// Output and execution error - PARSE OUTPUT + SYNTHETIC ERROR - Incomplete Run

	failSuite := func(msg, resultType, traceback string) core.TestSuite {
		return core.TestSuite{
			TestCases: []core.TestCase{
				{
					Name: target.Results.Name,
					Executions: []core.TestExecution{
						{
							Duration: &duration,
							Stdout:   stdout,
							Stderr:   stderr,
							Error: &core.TestResultFailure{
								Message:   msg,
								Type:      resultType,
								Traceback: traceback,
							},
						},
					},
				},
			},
		}
	}

	if len(resultsData) == 0 {
		if runError == nil {
			// No output and no execution error and output not expected - OK
			if target.NoTestOutput {
				return core.TestSuite{
					TestCases: []core.TestCase{
						{
							// Need a name so that multiple runs get collated correctly.
							Name: target.Results.Name,
							Executions: []core.TestExecution{
								{
									Duration: &duration,
									Stdout:   stdout,
									Stderr:   stderr,
								},
							},
						},
					},
				}
			}
			// No output and no execution error and output expected - SYNTHETIC ERROR - Missing Results
			return failSuite("Test failed to produce output results file", "MissingResults", "")
		}
		return failSuite("Test failed", "TestFailed", runError.Error())
	}

	results, parseError := parseTestResults(resultsData)
	if parseError != nil {
		// Output fails to parse with execution error - SYNTHETIC ERROR + EXECUTION ERROR - Failed to parse output
		if runError != nil {
			return failSuite("Test failed with no results", "NoResults", runError.Error())
		}
		// Output fails to parse - SYNTHETIC ERROR - Failed to parse output
		return failSuite("Couldn't parse test output file", "NoResults", parseError.Error())
	}

	// Output and no execution error - PARSE OUTPUT - Ignore noTestOutput
	if runError != nil && results.Failures() == 0 {
		// Add a failure result to the test so it shows up in the final aggregation.
		results.Add(failSuite("Test returned nonzero but reported no errors", "ReturnValue", runError.Error()).TestCases...)
	} else if runError == nil && results.Failures() != 0 {
		results.Add(failSuite("Test returned 0 but still reported failures", "ReturnValue", "").TestCases...)
	}

	return results
}

// Parses the coverage output for a single target.
func parseCoverageFile(target *core.BuildTarget, coverageFile string, run int) *core.TestCoverage {
	coverage, err := parseTestCoverageFile(target, coverageFile, run)
	if err != nil {
		log.Errorf("Failed to parse coverage file for %s: %s", target.Label, err)
	}
	return coverage
}

// RemoveTestOutputs removes any cached test or coverage result files for a target.
func RemoveTestOutputs(target *core.BuildTarget) error {
	if err := os.RemoveAll(target.TestResultsFile()); err != nil {
		return err
	} else if err := os.RemoveAll(target.CoverageFile()); err != nil {
		return err
	}
	for _, output := range target.TestOutputs {
		if err := os.RemoveAll(path.Join(target.OutDir(), output)); err != nil {
			return err
		}
	}
	return nil
}

// moveOutputFile moves an output file from the temporary directory to its permanent location.
// If dummy is given, it writes that into the destination if the file doesn't exist.
func moveOutputFile(state *core.BuildState, hash []byte, from, to, dummy string) error {
	if err := fs.EnsureDir(to); err != nil {
		return err
	}
	if !core.PathExists(from) {
		if dummy == "" {
			return nil
		}
		if err := ioutil.WriteFile(to, []byte(dummy), 0644); err != nil {
			return err
		}
	} else if err := os.Rename(from, to); err != nil {
		return err
	}
	// Set the hash on the new destination file
	return fs.RecordAttr(to, hash, xattrName, state.XattrsSupported)
}

// startTestWorkerIfNeeded starts a worker server if the test needs one.
func startTestWorkerIfNeeded(tid int, state *core.BuildState, target *core.BuildTarget) error {
	workerCmd, _, testCmd, err := core.TestWorkerCommand(state, target)
	if err != nil {
		return err
	} else if workerCmd == "" {
		return nil
	}
	state.LogBuildResult(tid, target.Label, core.TargetTesting, "Starting test worker...")
	resp, err := worker.EnsureWorkerStarted(state, workerCmd, testCmd, target)
	if err == nil {
		state.LogBuildResult(tid, target.Label, core.TargetTesting, "Testing...")
		if resp.Command != "" {
			log.Debug("Setting test command for %s to %s", target.Label, resp.Command)
			target.TestCommand = resp.Command
		}
	}
	return err
}

// verifyHash verifies that the hash on a test file matches the one for the current test.
func verifyHash(state *core.BuildState, filename string, hash []byte) bool {
	return bytes.Equal(hash, fs.ReadAttr(filename, xattrName, state.XattrsSupported))
}

// runtimeHash returns the runtime hash of a target, or an empty slice if running remotely.
func runtimeHash(tid int, state *core.BuildState, target *core.BuildTarget, runRemotely bool, run int) ([]byte, error) {
	if runRemotely {
		return nil, nil
	}
	if target.Local {
		// If the test is marked to run locally, download the inputs as we need these to calculate the runtime hash.
		if err := state.DownloadInputsIfNeeded(tid, target, true); err != nil {
			return nil, err
		}
	}
	hash, err := build.RuntimeHash(state, target, run)
	if err == nil {
		hash = core.CollapseHash(hash)
	}
	return hash, err
}
