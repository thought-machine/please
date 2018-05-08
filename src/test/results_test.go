package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"core"
)

func TestGoFailure(t *testing.T) {
	results, err := parseTestResults(new(core.BuildTarget), "src/test/test_data/go_test_failure.txt", false)
	require.NoError(t, err)
	assert.Equal(t, 4, results.NumTests)
	assert.Equal(t, 2, results.Passed)
	assert.Equal(t, 2, results.Failed)
	assert.Equal(t, 0, results.Skipped)
	assert.Equal(t, 0, results.ExpectedFailures)
}

func TestGoPassed(t *testing.T) {
	results, err := parseTestResults(new(core.BuildTarget), "src/test/test_data/go_test_pass.txt", false)
	require.NoError(t, err)
	assert.Equal(t, 4, results.NumTests)
	assert.Equal(t, 4, results.Passed)
	assert.Equal(t, 0, results.Failed)
	assert.Equal(t, 0, results.Skipped)
	assert.Equal(t, 0, results.ExpectedFailures)
}

func TestGoMultipleFailure(t *testing.T) {
	results, err := parseTestResults(new(core.BuildTarget), "src/test/test_data/go_multiple_failure.txt", false)
	require.NoError(t, err)
	assert.Equal(t, 2, results.NumTests)
	assert.Equal(t, 0, results.Passed)
	assert.Equal(t, 2, results.Failed)
	assert.Equal(t, 0, results.Skipped)
	assert.Equal(t, 0, results.ExpectedFailures)
}

func TestGoSkipped(t *testing.T) {
	results, err := parseTestResults(new(core.BuildTarget), "src/test/test_data/go_test_skip.txt", false)
	require.NoError(t, err)
	assert.Equal(t, 4, results.NumTests)
	assert.Equal(t, 3, results.Passed)
	assert.Equal(t, 0, results.Failed)
	assert.Equal(t, 1, results.Skipped)
	assert.Equal(t, 0, results.ExpectedFailures)
}

func TestGoSubtests(t *testing.T) {
	results, err := parseTestResults(new(core.BuildTarget), "src/test/test_data/go_subtests.txt", false)
	require.NoError(t, err)
	assert.Equal(t, 7, results.NumTests)
	assert.Equal(t, 7, results.Passed)
}

func TestBuckXML(t *testing.T) {
	results, err := parseTestResults(new(core.BuildTarget), "src/test/test_data/junit.xml", false)
	require.NoError(t, err)
	assert.Equal(t, 4, results.NumTests)
	assert.Equal(t, 4, results.Passed)
	assert.Equal(t, 0, results.Failed)
	assert.Equal(t, 0, results.Skipped)
	assert.Equal(t, 0, results.ExpectedFailures)
}

func TestJUnitXML(t *testing.T) {
	results, err := parseTestResults(new(core.BuildTarget), "src/test/test_data/xmlrunner-junit.xml", false)
	require.NoError(t, err)
	assert.Equal(t, 2, results.NumTests)
	assert.Equal(t, 1, results.Passed)
	assert.Equal(t, 1, results.Failed)
	assert.Equal(t, 0, results.Skipped)
	assert.Equal(t, 0, results.ExpectedFailures)
}

func TestKarmaXML(t *testing.T) {
	results, err := parseTestResults(new(core.BuildTarget), "src/test/test_data/karma-junit.xml", false)
	require.NoError(t, err)
	assert.Equal(t, 10, results.NumTests)
	assert.Equal(t, 10, results.Passed)
	assert.Equal(t, 0, results.Failed)
	assert.Equal(t, 0, results.Skipped)
	assert.Equal(t, 0, results.ExpectedFailures)
}

func TestUnitTestXML(t *testing.T) {
	results, err := parseTestResults(new(core.BuildTarget), "src/test/test_data/unittest.xml", false)
	require.NoError(t, err)
	assert.Equal(t, 2, results.NumTests)
	assert.Equal(t, 0, results.Passed)
	assert.Equal(t, 2, results.Failed)
	assert.Equal(t, 0, results.Skipped)
	assert.Equal(t, 0, results.ExpectedFailures)
}

func TestSkip(t *testing.T) {
	results, err := parseTestResults(new(core.BuildTarget), "src/test/test_data/xmlrunner-skipped.xml", false)
	require.NoError(t, err)
	assert.Equal(t, 2, results.NumTests)
	assert.Equal(t, 1, results.Passed)
	assert.Equal(t, 1, results.Skipped)
}

func TestGoSuite(t *testing.T) {
	results, err := parseTestResults(new(core.BuildTarget), "src/test/test_data/go_test_suite.txt", false)
	require.NoError(t, err)
	assert.Equal(t, 7, results.NumTests)
	assert.Equal(t, 5, results.Passed)
	assert.Equal(t, 1, results.Failed)
	assert.Equal(t, 1, results.Skipped)
	assert.Equal(t, 0, results.ExpectedFailures)
}

func TestGoIgnoreUnknownOutput(t *testing.T) {
	results, err := parseTestResults(new(core.BuildTarget), "src/test/test_data/go_test_ignore_logs.txt", false)
	require.NoError(t, err)
	assert.Equal(t, 4, results.NumTests)
	assert.Equal(t, 4, results.Passed)
	assert.Equal(t, 0, results.Failed)
	assert.Equal(t, 0, results.Skipped)
	assert.Equal(t, 0, results.ExpectedFailures)
}

func TestGoFailIfUnknownTestPasses(t *testing.T) {
	_, err := parseTestResults(new(core.BuildTarget), "src/test/test_data/go_test_unknown_test.txt", false)
	assert.Error(t, err)
}

func TestParseGoFileWithNoTests(t *testing.T) {
	_, err := parseTestResults(new(core.BuildTarget), "src/test/test_data/go_empty_test.txt", false)
	assert.NoError(t, err)
}

func TestParseGoFileWithLogging(t *testing.T) {
	results, err := parseTestResults(new(core.BuildTarget), "src/test/test_data/go_test_logging.txt", false)
	assert.NoError(t, err)
	assert.Equal(t, 3, results.NumTests)
	assert.Equal(t, 3, results.Passed)
	assert.Equal(t, 0, results.Failed)
}
