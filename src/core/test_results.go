package core

import (
	"bytes"
	"sort"
	"strings"
	"time"
)

// TestResults describes a set of test results for a test target.
type TestResults struct {
	NumTests         int // Total number of test cases in the test target.
	Passed           int // Number of tests that passed outright.
	Failed           int // Number of tests that failed.
	ExpectedFailures int // Number of tests that were expected to fail (counts as a pass, but displayed differently)
	Skipped          int // Number of tests skipped (also count as passes)
	Flakes           int // Number of failed attempts to run the test
	Results          []TestResult
	Output           string        // Stdout / stderr from the test.
	Cached           bool          // True if the test results were retrieved from cache
	TimedOut         bool          // True if the test failed because we timed it out.
	Duration         time.Duration // Length of time this test took
}

// TestResult represents detailed information about a test result
type TestResult struct {
	Name      string        // Name of failed test
	Type      string        // Type of failure, eg. type of exception raised
	Traceback string        // Traceback
	Stdout    string        // Standard output during test
	Stderr    string        // Standard error during test
	Duration  time.Duration // Time the test took
	Success   bool          // True if the test was successful
	Skipped   bool          // True if the test was skipped
}

// Aggregate aggregates the given results into this one.
func (results *TestResults) Aggregate(r *TestResults) {
	results.NumTests += r.NumTests
	results.Passed += r.Passed
	results.Failed += r.Failed
	results.ExpectedFailures += r.ExpectedFailures
	results.Skipped += r.Skipped
	results.Flakes += r.Flakes
	results.Results = append(results.Results, r.Results...)
	results.Duration += r.Duration
	// Output can't really be aggregated sensibly.
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
