package core

import (
	"bytes"
	"sort"
	"strings"
	"time"
)

// TestSuite describes all the test results for a target.
type TestSuite struct {
	Name      string     // The name of the test suite (usually the target name)
	Cached    bool       // True if the test results were retrieved from cache
	TimedOut  bool       // True if the test failed because we timed it out.
	TestCases []TestCase // The test cases that ran during this target
}

func (testSuite *TestSuite) Errors() uint {
	errors := uint(0)

	for _, result := range testSuite.TestCases {
		for _, execution := range result.Executions {
			if execution.Error != nil {
				errors++
			}
		}
	}

	return errors
}

func (testSuite *TestSuite) Failures() uint {
	failures := uint(0)

	for _, result := range testSuite.TestCases {
		for _, execution := range result.Executions {
			if execution.Failure != nil {
				failures++
			}
		}
	}

	return failures
}

func (testSuite *TestSuite) Skips() uint {
	skips := uint(0)

	for _, result := range testSuite.TestCases {
		for _, execution := range result.Executions {
			if execution.Skip != nil {
				skips++
			}
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

// TestExecution represents one execution of a test class.
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
