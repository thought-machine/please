package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/tools/cover"

	"github.com/thought-machine/please/src/core"
)

var target = &core.BuildTarget{Label: core.BuildLabel{PackageName: "src/test", Name: "coverage_test"}}

const (
	pythonCoverageFile    = "src/test/test_data/python-coverage.xml"
	goCoverageFile        = "src/test/test_data/go_coverage.txt"
	goCoverageFile2       = "src/test/test_data/go_coverage_2.txt"
	goCoverageFile3       = "src/test/test_data/go_coverage_3.txt"
	gcovCoverageFile      = "src/test/test_data/gcov_coverage.gcov"
	istanbulCoverageFile  = "src/test/test_data/istanbul_coverage.json"
	istanbulCoverageFile2 = "src/test/test_data/istanbul_coverage_2.json"
)

// Test that tests aren't required to produce coverage, ie. it's not an error if the file doesn't exist.
func TestCoverageNotRequired(t *testing.T) {
	coverage, err := parseTestCoverage(target, "src/test/test_data/blah.xml")
	if err != nil {
		t.Errorf("Incorrectly produced error attempting to read missing coverage file: %s", err)
	}
	if len(coverage.Files) != 0 {
		t.Errorf("Incorrectly reported some coverage results when there should be none.")
	}
}

// Test that the target is recorded in the file list.
func TestTargetIsRecorded(t *testing.T) {
	coverage, err := parseTestCoverage(target, pythonCoverageFile)
	if err != nil {
		t.Errorf("Failed to read coverage file %s", pythonCoverageFile)
	}
	if len(coverage.Tests) != 1 {
		t.Errorf("Expected exactly one test label recorded (got %d)", len(coverage.Tests))
	}
}

// Test the sample Python test output file.
func TestPythonResults(t *testing.T) {
	coverage, err := parseTestCoverage(target, pythonCoverageFile)
	if err != nil {
		t.Errorf("Failed to read coverage file %s", pythonCoverageFile)
	}
	if len(coverage.Files) != 4 {
		t.Errorf("Expected exactly four files covered by this test")
	}
	lines, present := coverage.Files["src/build/python/pex_test.py"]
	if !present {
		t.Errorf("Coverage info for src/build/python/pex_test.py not recorded.")
	}
	if len(lines) != 24 {
		t.Errorf("Expected exactly 24 lines of coverage information, was %d.", len(lines))
	}
	outputStr := core.TestCoverageString(lines)
	expected := "NNCNNCNCNCNCNNUNCNNCNNCU"
	if outputStr != expected {
		t.Errorf("Incorrect coverage output; was %s, expected %s", outputStr, expected)
	}
}

// Test the sample Go test output file.
func TestGoResults(t *testing.T) {
	coverage, err := parseTestCoverage(target, goCoverageFile)
	if err != nil {
		t.Errorf("Failed to read coverage file %s", goCoverageFile)
	}
	if len(coverage.Files) != 7 {
		t.Errorf("Expected exactly seven files covered by this test")
	}
	lines, present := coverage.Files["src/core/file_label.go"]
	if !present {
		t.Errorf("Coverage info for src/core/file_label.go not recorded.")
	}
	if len(lines) != 55 {
		t.Errorf("Expected exactly 55 lines of coverage information, was %d.", len(lines))
	}
	outputStr := core.TestCoverageString(lines)
	expected := "NNNNNNNNNNNNNNNUUUNUUUNUUUNUUUNNNNNNNNNNUUUNUUUNUUUNUUU"
	if len(expected) != 55 {
		t.Errorf("oops, expected string is wrong")
	}
	if outputStr != expected {
		t.Errorf("Incorrect coverage output; was %s, expected %s", outputStr, expected)
	}
}

// Test another sample Go file which has been observed to be wrong.
func TestGoResults2(t *testing.T) {
	coverage, err := parseTestCoverage(target, goCoverageFile2)
	if err != nil {
		t.Errorf("Failed to read coverage file %s", goCoverageFile2)
	}
	if len(coverage.Files) != 1 {
		t.Errorf("Expected exactly one file covered by this test")
	}
	lines, present := coverage.Files["src/core/state.go"]
	if !present {
		t.Errorf("Coverage info for src/core/file_label.go not recorded.")
	}
	if len(lines) != 273 {
		t.Errorf("Expected exactly 273 lines of coverage information, was %d.", len(lines))
	}

	assertLine(t, lines, 231, core.NotExecutable)
	assertLine(t, lines, 232, core.Covered)
	assertLine(t, lines, 233, core.Covered)
	assertLine(t, lines, 234, core.Covered)
	assertLine(t, lines, 235, core.Covered)
	assertLine(t, lines, 236, core.Covered)
	assertLine(t, lines, 237, core.Covered)
	assertLine(t, lines, 238, core.Covered)
	assertLine(t, lines, 239, core.Covered)
	assertLine(t, lines, 240, core.Covered)
}

func TestGoResults3(t *testing.T) {
	coverage, err := parseTestCoverage(target, goCoverageFile3)
	if err != nil {
		t.Errorf("Failed to read coverage file %s", goCoverageFile3)
	}
	if len(coverage.Files) != 1 {
		t.Errorf("Expected exactly one file covered by this test")
	}
	lines, present := coverage.Files["src/misc/plz_diff_graphs.go"]
	if !present {
		t.Errorf("Coverage info for src/misc/plz_diff_graphs.go not recorded.")
	}
	if len(lines) != 127 {
		t.Errorf("Expected exactly 127 lines of coverage information, was %d.", len(lines))
	}

	assertLine(t, lines, 67, core.NotExecutable)
	assertLine(t, lines, 68, core.Covered)
	assertLine(t, lines, 69, core.Covered)
	assertLine(t, lines, 81, core.NotExecutable)
}

// Direct test on the block-parsing function.
func TestParseBlocks(t *testing.T) {
	lines := parseBlocks([]cover.ProfileBlock{
		{StartLine: 2, EndLine: 3, Count: 1},
	})
	if len(lines) != 3 {
		t.Errorf("Wrong number of lines, should have been %d, was %d", 3, len(lines))
	}
	assertLine(t, lines, 1, core.NotExecutable)
	assertLine(t, lines, 2, core.Covered)
	assertLine(t, lines, 3, core.Covered)
}

func assertLine(t *testing.T, lines []core.LineCoverage, i int, expected core.LineCoverage) {
	i-- // 1-indexed
	if lines[i] != expected {
		t.Errorf("Line %d incorrect, should be %d, was %d", i, expected, lines[i])
	}
}

func TestGcovParsing(t *testing.T) {
	target := &core.BuildTarget{Label: core.BuildLabel{PackageName: "test", Name: "gcov_test"}}
	coverage, err := parseTestCoverage(target, gcovCoverageFile)
	assert.NoError(t, err)
	assert.Contains(t, coverage.Files, "test/cc_rules/deps_test.cc")
	lines := coverage.Files["test/cc_rules/deps_test.cc"]
	assertLine(t, lines, 1, core.NotExecutable)
	assertLine(t, lines, 2, core.NotExecutable)
	assertLine(t, lines, 9, core.NotExecutable)
	assertLine(t, lines, 10, core.Covered)
	assertLine(t, lines, 11, core.Covered)
	assertLine(t, lines, 12, core.Covered)
	assertLine(t, lines, 13, core.NotExecutable)
	assertLine(t, lines, 14, core.Covered)
	assertLine(t, lines, 15, core.Covered)
	assertLine(t, lines, 16, core.Covered)
	assertLine(t, lines, 17, core.NotExecutable)
	assertLine(t, lines, 18, core.Covered)
}

func TestIstanbulCoverage(t *testing.T) {
	target := &core.BuildTarget{Label: core.BuildLabel{PackageName: "common/js/components/ActionButton", Name: "test"}}
	coverage, err := parseTestCoverage(target, istanbulCoverageFile)
	assert.NoError(t, err)
	assert.Contains(t, coverage.Files, "common/js/components/ActionButton/ActionButton.js")
	assert.Contains(t, coverage.Files, "common/js/components/LoadingSpinner/LoadingSpinner.js")
	lines := coverage.Files["common/js/components/LoadingSpinner/LoadingSpinner.js"]
	assertLine(t, lines, 3, core.NotExecutable)
	assertLine(t, lines, 4, core.Covered)
	assertLine(t, lines, 5, core.Covered)
	assertLine(t, lines, 6, core.Covered)
	assertLine(t, lines, 7, core.Covered)
	assertLine(t, lines, 8, core.NotExecutable)
	assertLine(t, lines, 22, core.NotExecutable)
	assertLine(t, lines, 23, core.Uncovered)
	assertLine(t, lines, 24, core.NotExecutable)
}

func TestIstanbulCoverage2(t *testing.T) {
	target := &core.BuildTarget{Label: core.BuildLabel{PackageName: "common/js/components/Table", Name: "test"}}
	coverage, err := parseTestCoverage(target, istanbulCoverageFile2)
	assert.NoError(t, err)
	assert.Contains(t, coverage.Files, "common/js/components/Table/Table.js")
	lines := coverage.Files["common/js/components/Table/Table.js"]
	// This exercises a slightly more complex example with multiple overlapping statements.
	assertLine(t, lines, 15, core.Covered)
	assertLine(t, lines, 16, core.Covered)
	assertLine(t, lines, 17, core.Covered)
	assertLine(t, lines, 18, core.Covered)
	assertLine(t, lines, 19, core.Covered)
	assertLine(t, lines, 20, core.Covered)
	assertLine(t, lines, 21, core.Covered)
	assertLine(t, lines, 22, core.Covered)
	assertLine(t, lines, 23, core.Covered)
}
