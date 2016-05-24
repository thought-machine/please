package test

import "testing"

import "core"

func TestGoFailure(t *testing.T) {
	results, err := parseTestResults(new(core.BuildTarget), "src/test/test_data/go_test_failure.txt", false)
	if err != nil {
		t.Errorf("Unable to parse file: %s", err)
		return
	}
	assert(t, results.NumTests, 4, "tests")
	assert(t, results.Passed, 2, "passes")
	assert(t, results.Failed, 2, "failures")
	assert(t, results.Skipped, 0, "skipped tests")
	assert(t, results.ExpectedFailures, 0, "expected failures")
}

func TestGoPassed(t *testing.T) {
	results, err := parseTestResults(new(core.BuildTarget), "src/test/test_data/go_test_pass.txt", false)
	if err != nil {
		t.Errorf("Unable to parse file: %s", err)
		return
	}
	assert(t, results.NumTests, 4, "tests")
	assert(t, results.Passed, 4, "passes")
	assert(t, results.Failed, 0, "failures")
	assert(t, results.Skipped, 0, "skipped tests")
	assert(t, results.ExpectedFailures, 0, "expected failures")
}

func TestGoMultipleFailure(t *testing.T) {
	results, err := parseTestResults(new(core.BuildTarget), "src/test/test_data/go_multiple_failure.txt", false)
	if err != nil {
		t.Errorf("Unable to parse file: %s", err)
		return
	}
	assert(t, results.NumTests, 2, "tests")
	assert(t, results.Passed, 0, "passes")
	assert(t, results.Failed, 2, "failures")
	assert(t, results.Skipped, 0, "skipped tests")
	assert(t, results.ExpectedFailures, 0, "expected failures")
}

func TestGoSkipped(t *testing.T) {
	results, err := parseTestResults(new(core.BuildTarget), "src/test/test_data/go_test_skip.txt", false)
	if err != nil {
		t.Errorf("Unable to parse file: %s", err)
		return
	}
	assert(t, results.NumTests, 4, "tests")
	assert(t, results.Passed, 3, "passes")
	assert(t, results.Failed, 0, "failures")
	assert(t, results.Skipped, 1, "skipped tests")
	assert(t, results.ExpectedFailures, 0, "expected failures")
}

func TestBuckXML(t *testing.T) {
	results, err := parseTestResults(new(core.BuildTarget), "src/test/test_data/junit.xml", false)
	if err != nil {
		t.Errorf("Unable to parse file: %s", err)
		return
	}
	assert(t, results.NumTests, 4, "tests")
	assert(t, results.Passed, 4, "passes")
	assert(t, results.Failed, 0, "failures")
	assert(t, results.Skipped, 0, "skipped tests")
	assert(t, results.ExpectedFailures, 0, "expected failures")
}

func TestJUnitXML(t *testing.T) {
	results, err := parseTestResults(new(core.BuildTarget), "src/test/test_data/xmlrunner-junit.xml", false)
	if err != nil {
		t.Errorf("Unable to parse file: %s", err)
		return
	}
	assert(t, results.NumTests, 2, "tests")
	assert(t, results.Passed, 1, "passes")
	assert(t, results.Failed, 1, "failures")
	assert(t, results.Skipped, 0, "skipped tests")
	assert(t, results.ExpectedFailures, 0, "expected failures")
}

func TestKarmaXML(t *testing.T) {
	results, err := parseTestResults(new(core.BuildTarget), "src/test/test_data/karma-junit.xml", false)
	if err != nil {
		t.Errorf("Unable to parse file: %s", err)
		return
	}
	assert(t, results.NumTests, 10, "tests")
	assert(t, results.Passed, 10, "passes")
	assert(t, results.Failed, 0, "failures")
	assert(t, results.Skipped, 0, "skipped tests")
	assert(t, results.ExpectedFailures, 0, "expected failures")
}

func TestUnitTestXML(t *testing.T) {
	results, err := parseTestResults(new(core.BuildTarget), "src/test/test_data/unittest.xml", false)
	if err != nil {
		t.Errorf("Unable to parse file: %s", err)
		return
	}
	assert(t, results.NumTests, 2, "tests")
	assert(t, results.Passed, 0, "passes")
	assert(t, results.Failed, 2, "failures")
	assert(t, results.Skipped, 0, "skipped tests")
	assert(t, results.ExpectedFailures, 0, "expected failures")
}

func TestGoSuite(t *testing.T) {
	results, err := parseTestResults(new(core.BuildTarget), "src/test/test_data/go_test_suite.txt", false)
	if err != nil {
		t.Errorf("Unable to parse file: %s", err)
		return
	}
	assert(t, results.NumTests, 6, "tests")
	assert(t, results.Passed, 4, "passes")
	assert(t, results.Failed, 1, "failures")
	assert(t, results.Skipped, 1, "skipped tests")
	assert(t, results.ExpectedFailures, 0, "expected failures")
}

func TestGoIgnoreUnknownOutput(t *testing.T) {
	results, err := parseTestResults(new(core.BuildTarget), "src/test/test_data/go_test_ignore_logs.txt", false)
	if err != nil {
		t.Errorf("Unable to parse file: %s", err)
		return
	}
	assert(t, results.NumTests, 4, "tests")
	assert(t, results.Passed, 4, "passes")
	assert(t, results.Failed, 0, "failures")
	assert(t, results.Skipped, 0, "skipped tests")
	assert(t, results.ExpectedFailures, 0, "expected failures")
}

func TestGoFailIfUnknownTestPasses(t *testing.T) {
	_, err := parseTestResults(new(core.BuildTarget), "src/test/test_data/go_test_unknown_test.txt", false)
	if err == nil {
		t.Errorf("Results should not be parsable.")
	}
}

func TestParseGoFileWithNoTests(t *testing.T) {
	_, err := parseTestResults(new(core.BuildTarget), "src/test/test_data/go_empty_test.txt", false)
	if err != nil {
		t.Errorf("Unable to parse file: %s", err)
	}
}

// because I'm already pining for self.assertEqual...
func assert(t *testing.T, actual int, expected int, description string) {
	if actual != expected {
		t.Errorf("Unexpected number of %s: should be %d, was %d", description, expected, actual)
	}
}
