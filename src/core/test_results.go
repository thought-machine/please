package core

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/thought-machine/please/src/fs"
)

// TestSuites describes a collection of test results for a set of targets.
type TestSuites struct {
	TestSuites []TestSuite // The test results for each separate target
}

// TestSuite describes all the test results for a target.
type TestSuite struct {
	Package    string            // The package name of the test suite (usually the first part of the target label).
	Name       string            // The name of the test suite (usually the last part of the target label).
	Cached     bool              // True if the test results were retrieved from cache.
	Duration   time.Duration     // The length of time it took to run this target (may be different from the sum of times of test cases).
	TimedOut   bool              // True if the test failed because we timed it out.
	TestCases  TestCases         // The test cases that ran during execution of this target.
	Properties map[string]string // The system properties at the time of the test.
	Timestamp  string            // ISO8601 formatted datetime when the test ran.
}

// JavaStyleName pretends we are using a language that has package names and classnames etc.
func (testSuite TestSuite) JavaStyleName() string {
	return fmt.Sprintf("%s.%s", testSuite.Package, testSuite.Name)
}

// Collapse adds the results of one test suite to the current one.
func (testSuite *TestSuite) Collapse(incoming TestSuite) {
	testSuite.TestCases = append(testSuite.TestCases, incoming.TestCases...)
	testSuite.Duration += incoming.Duration
	testSuite.TimedOut = testSuite.TimedOut || incoming.TimedOut
	if testSuite.Properties == nil {
		testSuite.Properties = make(map[string]string)
	}
	testSuite.Properties = addAll(testSuite.Properties, incoming.Properties)
}

func addAll(map1 map[string]string, map2 map[string]string) map[string]string {
	for k, v := range map2 {
		map1[k] = v
	}
	return map1
}

// Tests returns the number of TestCases.
func (testSuite *TestSuite) Tests() int {
	return len(testSuite.TestCases)
}

// FlakyPasses returns the number of TestCases which succeeded after some number of executions.
func (testSuite TestSuite) FlakyPasses() int {
	flakyPasses := 0

	for _, result := range testSuite.TestCases {
		if result.Success() != nil && len(result.Executions) > 1 {
			flakyPasses++
		}
	}

	return flakyPasses
}

// Passes returns the number of TestCases which succeeded (not skipped).
func (testSuite TestSuite) Passes() int {
	passes := 0

	for _, result := range testSuite.TestCases {
		if len(result.Failures()) == 0 && len(result.Errors()) == 0 && result.Skip() == nil {
			passes++
		}
	}

	return passes
}

// Errors returns the number of TestCases which did not succeed and returned some abnormal error.
func (testSuite *TestSuite) Errors() int {
	errors := 0

	for _, result := range testSuite.TestCases {
		// No success result, not skipped, some errors (don't care about the presence of failures)
		if result.Success() == nil && result.Skip() == nil && len(result.Errors()) > 0 {
			errors++
		}
	}

	return errors
}

// Failures returns the number of TestCases which did not succeed and returned some failure.
func (testSuite *TestSuite) Failures() int {
	failures := 0

	for _, result := range testSuite.TestCases {
		// No success result, not skipped, no errors, but some failures.
		if result.Success() == nil && result.Skip() == nil && len(result.Errors()) == 0 && len(result.Failures()) > 0 {
			failures++
		}
	}

	return failures
}

// Skips returns the number of TestCases that were skipped.
func (testSuite *TestSuite) Skips() int {
	skips := 0

	for _, result := range testSuite.TestCases {
		if result.Skip() != nil {
			skips++
		}
	}

	return skips
}

// Add puts test cases together if they have the same name and classname, allowing callers to treat
// multiple test cases as if they were merely multiple executions of the same test.
func (testSuite *TestSuite) Add(cases ...TestCase) {
	for _, testCase := range cases {
		idx := findMatchingTestCase(&testCase, &testSuite.TestCases)

		if idx >= 0 {
			testSuite.TestCases[idx].Executions = append(testSuite.TestCases[idx].Executions, testCase.Executions...)
		} else {
			testSuite.TestCases = append(testSuite.TestCases, testCase)
		}
	}
}

func findMatchingTestCase(testCase *TestCase, testCases *TestCases) int {
	for idx := range *testCases {
		originalTestCase := (*testCases)[idx]
		if originalTestCase.Name == testCase.Name && originalTestCase.ClassName == testCase.ClassName {
			return idx
		}
	}
	return -1
}

// TestCase describes a set of test results for a test method.
type TestCase struct {
	ClassName  string          // ClassName of test (optional, for languages that don't have classes)
	Name       string          // Name of test
	Executions []TestExecution // The results of executing the test, possibly multiple times
}

// Success returns either the successful execution of a test case, or nil if it was never successfully executed.
func (testCase *TestCase) Success() *TestExecution {
	for _, execution := range testCase.Executions {
		if execution.Failure == nil &&
			execution.Error == nil &&
			execution.Skip == nil {
			return &execution
		}
	}
	return nil
}

// Skip returns the either the skipped execution of a test case, or nil if it was never skipped.
func (testCase *TestCase) Skip() *TestExecution {
	for _, execution := range testCase.Executions {
		if execution.Skip != nil {
			return &execution
		}
	}
	return nil
}

// Failures returns all failing executions of a test case.
func (testCase *TestCase) Failures() []TestExecution {
	failures := make([]TestExecution, 0)
	for _, execution := range testCase.Executions {
		if execution.Failure != nil {
			failures = append(failures, execution)
		}
	}
	return failures
}

// Errors returns all abnormal executions of a test case.
func (testCase *TestCase) Errors() []TestExecution {
	errors := make([]TestExecution, 0)
	for _, execution := range testCase.Executions {
		if execution.Error != nil {
			errors = append(errors, execution)
		}
	}
	return errors
}

// Duration calculates how long the test case took to run to success or failure (or nil if skipped or abnormal exit).
func (testCase *TestCase) Duration() *time.Duration {
	if testCase.Success() != nil {
		return testCase.Success().Duration
	} else if failures := testCase.Failures(); len(failures) > 0 {
		return failures[0].Duration
	}
	// Unable to determine duration of this test case.
	return nil
}

// TestCases is named so we can add a method to it.
type TestCases []TestCase

// AllSucceeded checks that every test case either passed or was skipped.
func (testCases TestCases) AllSucceeded() bool {
	for _, testCase := range testCases {
		if testCase.Success() == nil && testCase.Skip() == nil {
			return false
		}
	}
	return true
}

// TestExecution represents one execution of a test method. The absence of a Failure, Error or Skip implies the test
// executed successfully.
type TestExecution struct {
	Failure  *TestResultFailure // The failure, if any, running the test (usually an assertion that failed)
	Error    *TestResultFailure // The error, if any, running the test (usually some other abnormal exit)
	Skip     *TestResultSkip    // The reason for skipping the test, if it was skipped
	Stdout   string             // Standard output during test
	Stderr   string             // Standard error during test
	Duration *time.Duration     // How long the test took (if it did not fail abnormally)
}

// TestResultFailure stores the information related to the failure - stack trace, exception type etc.
type TestResultFailure struct {
	Type      string // The type of error (e.g. "AssertionError")
	Message   string // The reason for error (e.g. "1 != 2")
	Traceback string // The trace of the error (if known)
}

// TestResultSkip stores the reason for skipping a test.
type TestResultSkip struct {
	Message string // The reason for skipping the test
}

// A LineCoverage represents a single line of coverage, which can be in one of several states.
// Note that Please doesn't support sub-line coverage at present.
type LineCoverage uint8

// Constants representing the states that a single line can be in for coverage.
const (
	NotExecutable LineCoverage = iota // Line isn't executable (eg. comment, blank)
	Unreachable                       // Line is executable but we've determined it can't be reached. So far not used.
	Uncovered                         // Line is executable but isn't covered.
	Covered                           // Line is executable and covered.
)

var lineCoverageOutput = [...]rune{'N', 'X', 'U', 'C'} // Corresponds to ordering of enum.

// TestCoverage implements a pretty simple coverage format; we record one int for each line
// stating what its coverage is.
type TestCoverage struct {
	Tests map[BuildLabel]map[string][]LineCoverage
	Files map[string][]LineCoverage
}

// Aggregate aggregates results from another coverage object into this one.
func (coverage *TestCoverage) Aggregate(cov *TestCoverage) {
	if coverage.Tests == nil {
		coverage.Tests = map[BuildLabel]map[string][]LineCoverage{}
	}
	if coverage.Files == nil {
		coverage.Files = map[string][]LineCoverage{}
	}

	// Assume that tests are independent (will currently always be the case).
	for label, c := range cov.Tests {
		coverage.Tests[label] = c
	}
	// Files are more complex since multiple tests can cover the same file.
	// We take the best result for each line from each test.
	for filename, c := range cov.Files {
		coverage.Files[filename] = MergeCoverageLines(coverage.Files[filename], c)
	}
}

// MergeCoverageLines merges two sets of coverage results together, taking
// the superset of the two results.
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

// OrderedFiles returns an ordered slice of all the files we have coverage information for.
// Note that files are ordered non-trivially such that each directory remains together.
func (coverage *TestCoverage) OrderedFiles() []string {
	files := make([]string, 0, len(coverage.Files))
	for file := range coverage.Files {
		if strings.HasPrefix(file, RepoRoot) {
			file = strings.TrimLeft(file[len(RepoRoot):], "/")
		}
		files = append(files, file)
	}
	fs.SortPaths(files)
	return files
}

// NewTestCoverage constructs and returns a new TestCoverage instance.
func NewTestCoverage() *TestCoverage {
	return &TestCoverage{
		Tests: map[BuildLabel]map[string][]LineCoverage{},
		Files: map[string][]LineCoverage{},
	}
}

// TestCoverageString produces a string representation of coverage for serialising to file so we don't
// expose the internal enum values (ordering is important so we may want to insert
// new ones later). This format happens to be the same as the one Phabricator uses,
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
