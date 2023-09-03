package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thought-machine/please/src/core"
)

func TestGoFailure(t *testing.T) {
	results, err := parseTestResultsFile("src/test/test_data/go_test_failure.txt")
	require.NoError(t, err)
	assert.Equal(t, 4, len(results.TestCases))
	assert.Equal(t, 2, results.Passes())
	assert.Equal(t, 2, results.Failures())
	assert.Equal(t, 0, results.Skips())
}

func TestGoPassed(t *testing.T) {
	results, err := parseTestResultsFile("src/test/test_data/go_test_pass.txt")
	require.NoError(t, err)
	assert.Equal(t, 4, len(results.TestCases))
	assert.Equal(t, 4, results.Passes())
	assert.Equal(t, 0, results.Failures())
	assert.Equal(t, 0, results.Skips())
}

func TestGoMultipleFailure(t *testing.T) {
	results, err := parseTestResultsFile("src/test/test_data/go_multiple_failure.txt")
	require.NoError(t, err)
	assert.Equal(t, 2, len(results.TestCases))
	assert.Equal(t, 0, results.Passes())
	assert.Equal(t, 2, results.Failures())
	assert.Equal(t, 0, results.Skips())
}

func TestGoSkipped(t *testing.T) {
	results, err := parseTestResultsFile("src/test/test_data/go_test_skip.txt")
	require.NoError(t, err)
	assert.Equal(t, 4, len(results.TestCases))
	assert.Equal(t, 3, results.Passes())
	assert.Equal(t, 0, results.Failures())
	assert.Equal(t, 1, results.Skips())
}

// TestGoSkipped tests the skip messages in versions of Go prior to 1.14.
func TestGoSkippedMessage(t *testing.T) {
	results, err := parseTestResultsFile("src/test/test_data/go_test_skip.txt")
	require.NoError(t, err)

	var skippedTC = getFirstSkippedTestCase(results)
	assert.Equal(t, "interactive_display_test.go:21: haven't written proper support for this yet", skippedTC.Executions[0].Skip.Message)
}

// Go 1.14 changes the ordering of skipped messages in Go tests
func TestGoSkippedMessage114(t *testing.T) {
	results, err := parseTestResultsFile("src/test/test_data/go_test_skip_1_14.txt")
	require.NoError(t, err)

	var skippedTC = getFirstSkippedTestCase(results)
	assert.Equal(t, "TestSomething: my_test.go:9: This test is skipped", skippedTC.Executions[0].Skip.Message)
}

func getFirstSkippedTestCase(ts core.TestSuite) *core.TestCase {
	for _, tc := range ts.TestCases {
		if tc.Skip() != nil {
			return &tc
		}
	}
	return nil
}

// TestGoFailedMessage tests the location of failed test output in versions of Go prior to 1.14.
func TestGoFailedTraceback(t *testing.T) {
	results, err := parseTestResultsFile("src/test/test_data/go_test_failure.txt")
	require.NoError(t, err)

	var failedTC = getFirstFailedTestCase(results)
	assert.Equal(t, "results_test.go:11: Unable to parse file: EOF", failedTC.Executions[0].Failure.Message)
}

// Go 1.14 changes the ordering of failed messages in Go tests
func TestGoFailedTracebackGo114(t *testing.T) {
	results, err := parseTestResultsFile("src/test/test_data/go_test_fail_1_14.txt")
	require.NoError(t, err)

	var failedTC = getFirstFailedTestCase(results)
	assert.Equal(t, "TestFail: my_test.go:17: This test is going to fail.", failedTC.Executions[0].Failure.Message)
}

func getFirstFailedTestCase(ts core.TestSuite) *core.TestCase {
	for _, tc := range ts.TestCases {
		if len(tc.Failures()) > 0 {
			return &tc
		}
	}
	return nil
}

func TestGoSubtests(t *testing.T) {
	results, err := parseTestResultsFile("src/test/test_data/go_subtests.txt")
	require.NoError(t, err)
	assert.Equal(t, 7, len(results.TestCases))
	assert.Equal(t, 7, results.Passes())
}

func TestBuckXML(t *testing.T) {
	t.Skip("This format matches nothing we generate or care about")
	results, err := parseTestResultsFile("src/test/test_data/junit.xml")
	require.NoError(t, err)
	assert.Equal(t, 4, len(results.TestCases))
	assert.Equal(t, 4, results.Passes())
	assert.Equal(t, 0, results.Failures())
	assert.Equal(t, 0, results.Skips())
}

func TestJUnitXML(t *testing.T) {
	results, err := parseTestResultsFile("src/test/test_data/xmlrunner-junit.xml")
	require.NoError(t, err)
	assert.Equal(t, 2, len(results.TestCases))
	assert.Equal(t, 1, results.Passes())
	assert.Equal(t, 1, results.Failures())
	assert.Equal(t, 0, results.Skips())
}

func TestKarmaXML(t *testing.T) {
	results, err := parseTestResultsFile("src/test/test_data/karma-junit.xml")
	require.NoError(t, err)
	assert.Equal(t, 10, len(results.TestCases))
	assert.Equal(t, 10, results.Passes())
	assert.Equal(t, 0, results.Failures())
	assert.Equal(t, 0, results.Skips())
}

func TestUnitTestXML(t *testing.T) {
	results, err := parseTestResultsFile("src/test/test_data/unittest.xml")
	require.NoError(t, err)
	assert.Equal(t, 2, len(results.TestCases))
	assert.Equal(t, 0, results.Passes())
	assert.Equal(t, 2, results.Failures())
	assert.Equal(t, 0, results.Skips())
}

func TestSkip(t *testing.T) {
	results, err := parseTestResultsFile("src/test/test_data/xmlrunner-skipped.xml")
	require.NoError(t, err)
	assert.Equal(t, 2, len(results.TestCases))
	assert.Equal(t, 1, results.Passes())
	assert.Equal(t, 1, results.Skips())
}

func TestGoSuite(t *testing.T) {
	results, err := parseTestResultsFile("src/test/test_data/go_test_suite.txt")
	require.NoError(t, err)
	assert.Equal(t, 7, len(results.TestCases))
	assert.Equal(t, 5, results.Passes())
	assert.Equal(t, 1, results.Failures())
	assert.Equal(t, 1, results.Skips())
}

func TestGoIgnoreUnknownOutput(t *testing.T) {
	results, err := parseTestResultsFile("src/test/test_data/go_test_ignore_logs.txt")
	require.NoError(t, err)
	assert.Equal(t, 4, len(results.TestCases))
	assert.Equal(t, 4, results.Passes())
	assert.Equal(t, 0, results.Failures())
	assert.Equal(t, 0, results.Skips())
}

func TestParseGoFileWithNoTests(t *testing.T) {
	_, err := parseTestResultsFile("src/test/test_data/go_empty_test.txt")
	assert.NoError(t, err)
}

func TestParseGoFileWithLogging(t *testing.T) {
	results, err := parseTestResultsFile("src/test/test_data/go_test_logging.txt")
	assert.NoError(t, err)
	assert.Equal(t, 3, len(results.TestCases))
	assert.Equal(t, 3, results.Passes())
	assert.Equal(t, 0, results.Failures())
}
