package test

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"gopkg.in/op/go-logging.v1"

	"build"
	"core"
	"metrics"
)

var log = logging.MustGetLogger("test")

const dummyOutput = "=== RUN DummyTest\n--- PASS: DummyTest (0.00s)\nPASS\n"
const dummyCoverage = "<?xml version=\"1.0\" ?><coverage></coverage>"

// Test runs the tests for a single target.
func Test(tid int, state *core.BuildState, label core.BuildLabel) {
	state.LogBuildResult(tid, label, core.TargetTesting, "Testing...")
	startTime := time.Now()
	target := state.Graph.TargetOrDie(label)
	test(tid, state.ForTarget(target), label, target)
	metrics.Record(target, time.Since(startTime))
}

func test(tid int, state *core.BuildState, label core.BuildLabel, target *core.BuildTarget) {
	hash, err := build.RuntimeHash(state, target)
	if err != nil {
		state.LogBuildError(tid, label, core.TargetTestFailed, err, "Failed to calculate target hash")
		return
	}
	// Check the cached output files if the target wasn't rebuilt.
	hash = core.CollapseHash(hash)
	hashStr := base64.RawURLEncoding.EncodeToString(hash)
	resultsFileName := fmt.Sprintf(".test_results_%s_%s", label.Name, hashStr)
	coverageFileName := fmt.Sprintf(".test_coverage_%s_%s", label.Name, hashStr)
	outputFile := path.Join(target.TestDir(), "test.results")
	coverageFile := path.Join(target.TestDir(), "test.coverage")
	cachedOutputFile := path.Join(target.OutDir(), resultsFileName)
	cachedCoverageFile := path.Join(target.OutDir(), coverageFileName)
	needCoverage := state.NeedCoverage && !target.NoTestOutput

	cachedTestResults := func() core.TestSuite {
		log.Debug("Not re-running test %s; got cached results.", label)
		coverage := parseCoverageFile(target, cachedCoverageFile)
		results, err := parseTestResults(cachedOutputFile)
		results.Cached = true
		if err != nil {
			state.LogBuildError(tid, label, core.TargetTestFailed, err, "Failed to parse cached test file %s", cachedOutputFile)
		} else if results.Failures() > 0 {
			panic("Test results with failures shouldn't be cached.")
		} else {
			logTestSuccess(state, tid, label, &results, &coverage)
		}
		return results
	}

	moveAndCacheOutputFiles := func(results *core.TestSuite, coverage *core.TestCoverage) bool {
		// Never cache test results when given arguments; the results may be incomplete.
		if len(state.TestArgs) > 0 {
			log.Debug("Not caching results for %s, we passed it arguments", label)
			return true
		}
		if err := moveAndCacheOutputFile(state, target, hash, outputFile, cachedOutputFile, resultsFileName, dummyOutput); err != nil {
			state.LogTestResult(tid, label, core.TargetTestFailed, results, coverage, err, "Failed to move test output file")
			return false
		}
		if needCoverage || core.PathExists(coverageFile) {
			if err := moveAndCacheOutputFile(state, target, hash, coverageFile, cachedCoverageFile, coverageFileName, dummyCoverage); err != nil {
				state.LogTestResult(tid, label, core.TargetTestFailed, results, coverage, err, "Failed to move test coverage file")
				return false
			}
		}
		for _, output := range target.TestOutputs {
			tmpFile := path.Join(target.TestDir(), output)
			outFile := path.Join(target.OutDir(), output)
			if err := moveAndCacheOutputFile(state, target, hash, tmpFile, outFile, output, ""); err != nil {
				state.LogTestResult(tid, label, core.TargetTestFailed, results, coverage, err, "Failed to move test output file")
				return false
			}
		}
		return true
	}

	needToRun := func() bool {
		if target.State() == core.Unchanged && core.PathExists(cachedOutputFile) {
			// Output file exists already and appears to be valid. We might still need to rerun though
			// if the coverage files aren't available.
			if needCoverage && !core.PathExists(cachedCoverageFile) {
				return true
			}
			return false
		}
		// Check the cache for these artifacts.
		if state.Cache == nil {
			return true
		}
		if !state.Cache.RetrieveExtra(target, hash, resultsFileName) {
			return true
		}
		if needCoverage && !state.Cache.RetrieveExtra(target, hash, coverageFileName) {
			return true
		}
		for _, output := range target.TestOutputs {
			if !state.Cache.RetrieveExtra(target, hash, output) {
				return true
			}
		}
		return false
	}

	// Don't cache when doing multiple runs, presumably the user explicitly wants to check it.
	if state.NumTestRuns <= 1 && !needToRun() {
		target.Results = cachedTestResults()
		return
	}

	// Fresh set of results for this target.
	target.Results = core.TestSuite{
		Name: toClassName(target.Label.String()),
	}

	// Remove any cached test result file.
	if err := RemoveCachedTestFiles(target); err != nil {
		state.LogBuildError(tid, label, core.TargetTestFailed, err, "Failed to remove cached test files")
		return
	}
	if worker, err := startTestWorkerIfNeeded(tid, state, target); err != nil {
		state.LogBuildError(tid, label, core.TargetTestFailed, err, "Failed to start test worker")
		testCase := core.TestCase{
			Name: worker,
			Executions: []core.TestExecution{
				{
					Failure: &core.TestResultFailure{
						Message:   "Failed to start test worker",
						Type:      "WorkerFail",
						Traceback: err.Error(),
					},
				},
			},
		}
		target.Results.TestCases = append(target.Results.TestCases, testCase)
		return
	}
	numSucceeded := 0
	numRuns, successesRequired := calcNumRuns(state.NumTestRuns, target.Flakiness)
	var resultErr error
	resultMsg := ""
	var coverage core.TestCoverage

	// TODO(agenticarus): Push the flakiness down into the test runners to speed them up?
	for i := 0; i < numRuns && numSucceeded < successesRequired; i++ {
		if numRuns > 1 {
			state.LogBuildResult(tid, label, core.TargetTesting, fmt.Sprintf("Testing (%d of %d)...", i+1, numRuns))
		}
		startTime := time.Now() // reset this for next time
		out, runError := prepareAndRunTest(tid, state, target)
		duration := time.Since(startTime)

		// This is all pretty involved; there are lots of different possibilities of what could happen.
		// The contract is that the test must return zero on success or non-zero on failure (Unix FTW).
		// If it's successful, it must produce a parseable file named "test.results" in its temp folder.
		// (alternatively, this can be a directory containing parseable files).
		// Tests can opt out of the file requirement individually, in which case they're judged only
		// by their return value.
		// But of course, we still have to consider all the alternatives here and handle them nicely.
		target.Results.Duration = duration
		target.Results.TimedOut = runError == context.DeadlineExceeded
		if !core.PathExists(outputFile) {
			if runError == nil && target.NoTestOutput {
				testCase := core.TestCase{
					Name: target.Results.Name,
					Executions: []core.TestExecution{
						{
							Duration: &duration,
							Stdout:   string(out),
						},
					},
				}
				target.Results.TestCases = append(target.Results.TestCases, testCase)
				numSucceeded++
			} else if runError == nil {
				testCase := core.TestCase{
					Name: target.Results.Name,
					Executions: []core.TestExecution{
						{
							Duration: &duration,
							Failure: &core.TestResultFailure{
								Message: "Missing results",
								Type:    "MissingResults",
							},
							Stdout: string(out),
						},
					},
				}
				target.Results.TestCases = append(target.Results.TestCases, testCase)
				resultErr = fmt.Errorf("Test failed to produce output results file")
				resultMsg = fmt.Sprintf("Test apparently succeeded but failed to produce %s. Output: %s", outputFile, string(out))
			} else {
				testCase := core.TestCase{
					Name: target.Results.Name,
					Executions: []core.TestExecution{
						{
							Duration: &duration,
							Error: &core.TestResultFailure{
								Message:   "Test errored with no results",
								Type:      "NoResults",
								Traceback: runError.Error(),
							},
							Stdout: string(out),
						},
					},
				}
				target.Results.TestCases = append(target.Results.TestCases, testCase)
				resultErr = runError
				resultMsg = fmt.Sprintf("Test failed with no results. Output: %s", string(out))
			}
		} else {
			results, parseError := parseTestResults(outputFile)
			for idx, _ := range results.TestCases {
				testCase := &results.TestCases[idx]
				if testCase.ClassName == "GoTest" {
					testCase.ClassName = fmt.Sprintf("%s.GoTest", toClassName(target.Label.String()))
				}
			}
			target.Results.Aggregate(results)
			if parseError != nil {
				resultErr = parseError
				resultMsg = fmt.Sprintf("Couldn't parse test output file: %s. Stdout: %s", parseError, string(out))
			} else if runError != nil && results.Failures() == 0 {
				// Add a failure result to the test so it shows up in the final aggregation.
				testCase := core.TestCase{
					// We don't know the type of test we ran :(
					ClassName: fmt.Sprintf("%s.Test", toClassName(target.Label.String())),
					Name: results.Name,
					Executions: []core.TestExecution{
						{
							Failure: &core.TestResultFailure{
								Type:    "ReturnValue",
								Message: fmt.Sprintf("%s", runError),
							},
							Duration: &duration,
							Stdout:   string(out),
						},
					},
				}
				target.Results.TestCases = append(target.Results.TestCases, testCase)
				resultErr = runError
				resultMsg = fmt.Sprintf("Test returned nonzero but reported no errors: %s. Output: %s", runError, string(out))
			} else if runError == nil && results.Failures() != 0 {
				resultErr = fmt.Errorf("Test returned 0 but still reported failures")
				resultMsg = fmt.Sprintf("Test returned 0 but still reported failures. Stdout: %s", string(out))
			} else if results.Failures() != 0 {
				resultErr = fmt.Errorf("Tests failed")
				resultMsg = fmt.Sprintf("Tests failed. Stdout: %s", string(out))
			} else {
				numSucceeded++
			}
		}
	}

	coverage = parseCoverageFile(target, coverageFile)

	if numSucceeded >= successesRequired {
		// Success, clean things up
		if moveAndCacheOutputFiles(&target.Results, &coverage) {
			logTestSuccess(state, tid, label, &target.Results, &coverage)
		}
		// Clean up the test directory.
		if state.CleanWorkdirs {
			if err := os.RemoveAll(target.TestDir()); err != nil {
				log.Warning("Failed to remove test directory for %s: %s", target.Label, err)
			}
		}
	} else {
		state.LogTestResult(tid, label, core.TargetTestFailed, &target.Results, &coverage, resultErr, resultMsg)
	}
}

// targetLabel is of the form //src/core:config_test
// So the "classname" is src.core
func toClassName(targetLabel string) string {
	return strings.Replace(strings.Replace(targetLabel[2:], "/", ".", -1), ":", ".", -1)
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

func pluralise(word string, quantity uint) string {
	if quantity == 1 {
		return word
	}
	return word + "s"
}

func prepareTestDir(graph *core.BuildGraph, target *core.BuildTarget) error {
	if err := os.RemoveAll(target.TestDir()); err != nil {
		return err
	}
	if err := os.MkdirAll(target.TestDir(), core.DirPermissions); err != nil {
		return err
	}
	for out := range core.IterRuntimeFiles(graph, target, true) {
		if err := core.PrepareSourcePair(out); err != nil {
			return err
		}
	}
	return nil
}

// testCommandAndEnv returns the test command & environment for a target.
func testCommandAndEnv(state *core.BuildState, target *core.BuildTarget) (string, []string) {
	replacedCmd := build.ReplaceTestSequences(state, target, target.GetTestCommand(state))
	env := core.TestEnvironment(state, target, path.Join(core.RepoRoot, target.TestDir()))
	if len(state.TestArgs) > 0 {
		args := strings.Join(state.TestArgs, " ")
		replacedCmd += " " + args
		env = append(env, "TESTS="+args)
	}
	return replacedCmd, env
}

func runTest(state *core.BuildState, target *core.BuildTarget) ([]byte, error) {
	replacedCmd, env := testCommandAndEnv(state, target)
	log.Debug("Running test %s\nENVIRONMENT:\n%s\n%s", target.Label, strings.Join(env, "\n"), replacedCmd)
	_, out, err := core.ExecWithTimeoutShellStdStreams(state, target, target.TestDir(), env, target.TestTimeout, state.Config.Test.Timeout, state.ShowAllOutput, replacedCmd, target.TestSandbox, state.DebugTests)
	return out, err
}

// prepareAndRunTest sets up a test directory and runs the test.
func prepareAndRunTest(tid int, state *core.BuildState, target *core.BuildTarget) (out []byte, err error) {
	if err = prepareTestDir(state.Graph, target); err != nil {
		state.LogBuildError(tid, target.Label, core.TargetTestFailed, err, "Failed to prepare test directory for %s: %s", target.Label, err)
		return []byte{}, err
	}
	return runPossiblyContainerisedTest(tid, state, target)
}

// Parses the coverage output for a single target.
func parseCoverageFile(target *core.BuildTarget, coverageFile string) core.TestCoverage {
	coverage, err := parseTestCoverage(target, coverageFile)
	if err != nil {
		log.Errorf("Failed to parse coverage file for %s: %s", target.Label, err)
	}
	return coverage
}

// RemoveCachedTestFiles removes any cached test or coverage result files for a target.
func RemoveCachedTestFiles(target *core.BuildTarget) error {
	if err := removeAnyFilesWithPrefix(target.OutDir(), ".test_results_"+target.Label.Name); err != nil {
		return err
	}
	if err := removeAnyFilesWithPrefix(target.OutDir(), ".test_coverage_"+target.Label.Name); err != nil {
		return err
	}
	for _, output := range target.TestOutputs {
		if err := os.RemoveAll(path.Join(target.OutDir(), output)); err != nil {
			return err
		}
	}
	return nil
}

// removeAnyFilesWithPrefix deletes any files in a directory matching a given prefix.
func removeAnyFilesWithPrefix(dir, prefix string) error {
	infos, err := ioutil.ReadDir(dir)
	if err != nil {
		// Not an error if the directory just isn't there.
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, info := range infos {
		if strings.HasPrefix(info.Name(), prefix) {
			if err := os.RemoveAll(path.Join(dir, info.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

// Attempt to write a dummy coverage file to record that it's been done for a test.
func moveAndCacheOutputFile(state *core.BuildState, target *core.BuildTarget, hash []byte, from, to, filename, dummy string) error {
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
	if state.Cache != nil {
		state.Cache.StoreExtra(target, hash, filename)
	}
	return nil
}

// calcNumRuns works out how many total runs we should have for a test, and how many successes
// are required for it to count as success.
// numRuns and flakiness default to 1 which mean run once.
func calcNumRuns(numRuns, flakiness int) (int, int) {
	return numRuns * flakiness, numRuns
}

// startTestWorkerIfNeeded starts a worker server if the test needs one.
func startTestWorkerIfNeeded(tid int, state *core.BuildState, target *core.BuildTarget) (string, error) {
	worker, _, _ := build.TestWorkerCommand(state, target)
	if worker == "" {
		return "", nil
	}
	state.LogBuildResult(tid, target.Label, core.TargetTesting, "Starting test worker...")
	err := build.EnsureWorkerStarted(state, worker, target.Label)
	if err == nil {
		state.LogBuildResult(tid, target.Label, core.TargetTesting, "Testing...")
	}
	return worker, err
}
