package core

import (
	"bytes"
	"sort"
	"strings"
	"time"
	"fmt"
)

// TestSuite describes all the test results for a target.
type TestSuite struct {
	Name      string     // The name of the test suite (usually the target name)
	Cached    bool       // True if the test results were retrieved from cache
	TimedOut  bool       // True if the test failed because we timed it out.
	TestCases []TestCase // The test cases that ran during this target
}

// Passes returns the number of TestCases which succeeded (not skipped).
func (suite TestSuite) Passes() uint {
	passes := uint(0)

	for _, result := range suite.TestCases {
		if result.Success() != nil {
			passes++
		}
	}

	return passes
}

// Errors returns the number of TestCases which did not succeed and returned some error.
func (testSuite *TestSuite) Errors() uint {
	errors := uint(0)

	for _, result := range testSuite.TestCases {
		// No success result, not skipped, some errors (don't care about the presence of failures)
		if result.Success() == nil && result.Skip() == nil && len(result.Errors()) > 0 {
			errors++
		}
	}

	return errors
}

// Failures returns the number of TestCases which did not succeed and returned some failure.
func (testSuite *TestSuite) Failures() uint {
	failures := uint(0)

	for _, result := range testSuite.TestCases {
		// No success result, not skipped, no errors, but some failures.
		if result.Success() == nil && result.Skip() == nil && len(result.Errors()) == 0 && len(result.Failures()) > 0 {
			failures++
		}
	}

	return failures
}

// Skips returns the number of TestCases that were skipped.
func (testSuite *TestSuite) Skips() uint {
	skips := uint(0)

	for _, result := range testSuite.TestCases {
		if result.Skip() != nil {
			skips++
		}
	}

	return skips
}

func (testSuite *TestSuite) Duration() time.Duration {
	duration := time.Duration(0)

	for _, result := range testSuite.TestCases {
		for _, execution := range result.Executions {
			if execution.Duration != nil {
				duration = duration + *execution.Duration
			}
		}
	}

	return duration
}

func (testSuite *TestSuite) Tests() uint {
	return uint(len(testSuite.TestCases))
}
func (testSuite *TestSuite) Aggregate(suite2 TestSuite) {
	extraTestCases := make([]TestCase, 0)

	OUTER:
	for idx2 := range suite2.TestCases {
		testCase2 := &suite2.TestCases[idx2]
		for idx1, _ := range testSuite.TestCases {
			testCase := &testSuite.TestCases[idx1]
			if testCase.ClassName == testCase2.ClassName && testCase.Name == testCase2.Name {
				testCase.Aggregate(testCase2)
				continue OUTER
			}
		}
		// No matching test case, just add it in afterwards
		extraTestCases = append(extraTestCases, *testCase2)
	}

	testSuite.TestCases = append(testSuite.TestCases, extraTestCases...)
}

// TestCase describes a set of test results for a test target.
type TestCase struct {
	ClassName  string          // ClassName of test (optional, for languages that don't have classes)
	Name       string          // Name of test
	Executions []TestExecution // The results of executing the test, possibly multiple times
}

func (testCase TestCase) Success() *TestExecution {
	for _, execution := range testCase.Executions {
		if execution.Failure == nil &&
			execution.Error == nil &&
			execution.Skip == nil {
			return &execution
		}
	}
	return nil
}

func (testCase TestCase) Skip() *TestExecution {
	for _, execution := range testCase.Executions {
		if execution.Skip != nil {
			return &execution
		}
	}
	return nil
}

func (testCase TestCase) Failures() []TestExecution {
	failures := make([]TestExecution, 0)
	for _, execution := range testCase.Executions {
		if execution.Failure != nil {
			failures = append(failures, execution)
		}
	}
	return failures
}

func (testCase TestCase) Errors() []TestExecution {
	errors := make([]TestExecution, 0)
	for _, execution := range testCase.Executions {
		if execution.Error != nil {
			errors = append(errors, execution)
		}
	}
	return errors
}

func (testCase TestCase) Duration() *time.Duration {
	if testCase.Success() != nil {
		return testCase.Success().Duration
	} else if len(testCase.Failures()) > 0{
		return testCase.Failures()[0].Duration
	}
	// Unable to determine duration of this test case.
	return nil
}

func (testCase *TestCase) Aggregate(testCase2 *TestCase) {
	if testCase.ClassName != testCase2.ClassName || testCase.Name != testCase2.Name {
		panic(fmt.Sprintf("Unable to aggregate testcases as classnames and test names are different"))
	}
	testCase.Executions = append(testCase.Executions, testCase2.Executions...)
}

// TestExecution represents one execution of a test class. The absence of a Failure, Error or Skip implies the test
// executed successfully.
type TestExecution struct {
	Failure  *TestResultFailure // The failure, if any, running the test (usually an assertion that failed)
	Error    *TestResultFailure // The error, if any, running the test (usually some other abnormal exit)
	Skip     *TestResultSkip    // The reason for skipping the test, if it was skipped
	Stdout   string             // Standard output during test
	Stderr   string             // Standard error during test
	Duration *time.Duration     // How long the test took (if it did not fail abnormally)
}

type TestResultFailure struct {
	Type      string // The type of error (e.g. "AssertionError")
	Message   string // The reason for error (e.g. "1 != 2")
	Traceback string // The trace of the error (if known)
}

type TestResultSkip struct {
	Message string // The reason for skipping the test
}

// A LineCoverage represents a single line of coverage, which can be in one of several states.
// Note that Please doesn't support sub-line coverage at present.
type LineCoverage uint8

// Constants representing the states that a single line can be in for coverage.
const (
	NotExecutable LineCoverage = iota // Line isn't executable (eg. comment, blank)
	Unreachable   LineCoverage = iota // Line is executable but we've determined it can't be reached. So far not used.
	Uncovered     LineCoverage = iota // Line is executable but isn't covered.
	Covered       LineCoverage = iota // Line is executable and covered.
)

var lineCoverageOutput = [...]rune{'N', 'X', 'U', 'C'} // Corresponds to ordering of enum.

// TestCoverage implements a pretty simple coverage format; we record one int for each line
// stating what its coverage is.
type TestCoverage struct {
	Tests map[BuildLabel]map[string][]LineCoverage
	Files map[string][]LineCoverage
}

// Aggregate aggregates results from that coverage object into this one.
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
func (coverage *TestCoverage) OrderedFiles() []string {
	files := []string{}
	for file := range coverage.Files {
		if strings.HasPrefix(file, RepoRoot) {
			file = strings.TrimLeft(file[len(RepoRoot):], "/")
		}
		files = append(files, file)
	}
	sort.Strings(files)
	return files
}

// NewTestCoverage constructs and returns a new TestCoverage instance.
func NewTestCoverage() TestCoverage {
	return TestCoverage{
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
