// Parser for JUnit XML output.

package test

import (
	"bytes"
	"encoding/xml"
	"io/ioutil"
	"os"
	"path"
	"time"

	"core"
)

func looksLikeJUnitXMLTestResults(b []byte) bool {
	return bytes.HasPrefix(b, []byte{'<', '?', 'x', 'm', 'l'}) || bytes.HasPrefix(b, []byte{'<', 't', 'e', 's', 't'})
}

func parseJUnitXMLTestResults(bytes []byte) (core.TestSuite, error) {
	results := core.TestSuite{}
	junitCase := jUnitXMLTestResults{}
	if err := xml.Unmarshal(bytes, &junitCase); err != nil {
		return results, err
	}
	if len(junitCase.Tests) > 0 {
		for _, test := range junitCase.Tests {
			result := core.TestCase{
				ClassName: test.ClassName,
				Name:      test.Name,
			}
			appendResult(test, &result)
			results.TestCases = append(results.TestCases, result)
		}
	}
	if len(junitCase.TestCases) > 0 {
		for _, test := range junitCase.TestCases {
			result := core.TestCase{
				ClassName: test.ClassName,
				Name:      test.Name,
			}
			appendResult(test, &result)
			results.TestCases = append(results.TestCases, result)
		}
	}
	for _, testSuite := range junitCase.TestSuites {
		for _, test := range testSuite.TestCases {
			result := core.TestCase{
				ClassName: test.ClassName,
				Name:      test.Name,
			}
			appendResult(test, &result)
			results.TestCases = append(results.TestCases, result)
		}
	}
	return results, nil
}

func appendResult(test jUnitXMLTest, results *core.TestCase) {
	// There can be only one of these
	if test.Failure != nil {
		appendFailure(test, results, *test.Failure)
	} else if test.Error != nil {
		appendError(test, results, *test.Error)
	} else if test.Skipped != nil {
		appendSkipped(test, results, *test.Skipped)
	} else {
		appendSuccess(test, results)
	}

	if len(test.FlakyFailure) > 0 {
		for _, flake := range test.FlakyFailure {
			appendFlakyFailure(test, results, flake)
		}
	}
	if len(test.FlakyError) > 0 {
		// The test ultimately succeeded but errored possibly several times.
		// We have added the success above.
		for _, flake := range test.FlakyError {
			appendFlakyError(test, results, flake)
		}
	}
	if len(test.RerunFailure) > 0 {
		// The test never succeeded and flaked possibly several times.
		// We have already added the first failure above.
		for _, flake := range test.RerunFailure {
			appendRerunFailure(test, results, flake)
		}
	}
	if len(test.RerunError) > 0 {
		// The test never succeeded and errored possibly several times.
		// We have already added the first error above.
		for _, flake := range test.RerunError {
			appendRerunError(test, results, flake)
		}
	}
}

func appendFailure(test jUnitXMLTest, results *core.TestCase, failure jUnitXMLFailure) {
	duration := failure.Duration()
	results.Executions = append(results.Executions, core.TestExecution{
		Failure: &core.TestResultFailure{
			Message:   failure.Message,
			Type:      failure.Type,
			Traceback: failure.Traceback,
		},
		Stdout:   test.Stdout,
		Stderr:   test.Stderr,
		Duration: &duration,
	})
}

func appendFlakyFailure(test jUnitXMLTest, results *core.TestCase, flake jUnitXMLFlaky) {
	results.Executions = append(results.Executions, core.TestExecution{
		Failure: &core.TestResultFailure{
			Message:   flake.Message,
			Type:      flake.Type,
			Traceback: flake.Traceback,
		},
		Stdout: test.Stdout,
		Stderr: test.Stderr,
	})
}

func appendFlakyError(test jUnitXMLTest, results *core.TestCase, flake jUnitXMLFlaky) {
	results.Executions = append(results.Executions, core.TestExecution{
		Error: &core.TestResultFailure{
			Message:   flake.Message,
			Type:      flake.Type,
			Traceback: flake.Traceback,
		},
		Stdout: test.Stdout,
		Stderr: test.Stderr,
	})
}

func appendRerunFailure(test jUnitXMLTest, results *core.TestCase, flake jUnitXMLRerunFailure) {
	results.Executions = append(results.Executions, core.TestExecution{
		Failure: &core.TestResultFailure{
			Message:   flake.Message,
			Type:      flake.Type,
			Traceback: flake.Traceback,
		},
		Stdout: test.Stdout,
		Stderr: test.Stderr,
	})
}

func appendRerunError(test jUnitXMLTest, results *core.TestCase, flake jUnitXMLRerunError) {
	results.Executions = append(results.Executions, core.TestExecution{
		Error: &core.TestResultFailure{
			Message:   flake.Message,
			Type:      flake.Type,
			Traceback: flake.Traceback,
		},
		Stdout: test.Stdout,
		Stderr: test.Stderr,
	})
}

func appendError(test jUnitXMLTest, results *core.TestCase, error jUnitXMLError) {
	results.Executions = append(results.Executions, core.TestExecution{
		Error: &core.TestResultFailure{
			Message:   error.Message,
			Type:      error.Type,
			Traceback: error.Traceback,
		},
		Stdout: test.Stdout,
		Stderr: test.Stderr,
	})
}

func appendSkipped(test jUnitXMLTest, results *core.TestCase, skipped jUnitXMLSkipped) {
	results.Executions = append(results.Executions, core.TestExecution{
		Skip: &core.TestResultSkip{
			Message: skipped.Message,
		},
		Stdout: test.Stdout,
		Stderr: test.Stderr,
	})
}

func appendSuccess(test jUnitXMLTest, results *core.TestCase) {
	duration := test.Duration()
	results.Executions = append(results.Executions, core.TestExecution{
		Duration: &duration,
		Stdout:   test.Stdout,
		Stderr:   test.Stderr,
	})
}

type jUnitXMLTestResults struct {
	TestSuites []jUnitXMLTestSuite `xml:"testsuite"`
	TestCases  []jUnitXMLTest      `xml:"testcase"`
	Tests      []jUnitXMLTest      `xml:"test"`
	XMLName    xml.Name
}

type jUnitXMLTestSuite struct {
	Name     string `xml:"name,attr"`
	Errors   uint   `xml:"errors,attr"`
	Failures uint   `xml:"failures,attr"`
	Group    string `xml:"group,attr,omitempty"`
	Skipped  uint   `xml:"skipped,attr"`
	Tests    uint   `xml:"tests,attr"`
	*Timed          `xml:"time,attr,omitempty"`

	Properties jUnitXMLProperties `xml:"properties,omitempty"`
	TestCases  []jUnitXMLTest     `xml:"testcase"`
}

type jUnitXMLTest struct {
	ClassName string `xml:"classname,attr,omitempty"`
	Name      string `xml:"name,attr"`
	Group     string `xml:"group,attr,omitempty"`
	Timed            `xml:"time,attr"`

	Error        *jUnitXMLError         `xml:"error,omitempty"`
	FlakyError   []jUnitXMLFlaky        `xml:"flakyError,omitempty"`
	RerunError   []jUnitXMLRerunError   `xml:"rerunError,omitempty"`
	Failure      *jUnitXMLFailure       `xml:"failure,omitempty"`
	FlakyFailure []jUnitXMLFlaky        `xml:"flakyFailure,omitempty"`
	RerunFailure []jUnitXMLRerunFailure `xml:"rerunFailure,omitempty"`
	Skipped      *jUnitXMLSkipped       `xml:"skipped,omitempty"`
	Stdout       string                 `xml:"system-out,omitempty"`
	Stderr       string                 `xml:"system-err,omitempty"`
}

type jUnitXMLProperties struct {
	Property []jUnitXMLProperty `xml:"property"`
}

type jUnitXMLProperty struct {
	Name  string `xml:"name"`
	Value string `xml:"value"`
}

type jUnitXMLError struct {
	Message string `xml:"message,attr,omitempty"`
	Type    string `xml:"type,attr"`

	Traceback string `xml:"chardata"`
}

type jUnitXMLFailure struct {
	Message string `xml:"message,attr,omitempty"`
	Timed          `xml:"time,attr"`
	Type    string `xml:"type,attr"`

	Traceback string `xml:"chardata"`
}

type jUnitXMLFlaky struct {
	Message string `xml:"message,attr,omitempty"`
	Type    string `xml:"type,attr"`

	Traceback string `xml:"chardata"`
	Stdout    string `xml:"system-out,omitempty"`
	Stderr    string `xml:"system-err,omitempty"`
}

type jUnitXMLRerunError struct {
	Message string `xml:"message,attr,omitempty"`
	Type    string `xml:"type,attr"`

	Traceback string `xml:"chardata"`
	Stdout    string `xml:"system-out,omitempty"`
	Stderr    string `xml:"system-err,omitempty"`
}

type jUnitXMLRerunFailure struct {
	Message string `xml:"message,attr,omitempty"`
	Timed          `xml:"time,attr"`
	Type    string `xml:"type,attr"`

	Traceback string `xml:"chardata"`
	Stdout    string `xml:"system-out,omitempty"`
	Stderr    string `xml:"system-err,omitempty"`
}

type jUnitXMLSkipped struct {
	Message string `xml:"message,attr,omitempty"`
}

type Timed struct {
	Time float64 `xml:"time,attr"`
}

func (j Timed) Duration() time.Duration {
	return time.Duration(j.Time * float64(time.Second))
}

func (j jUnitXMLTest) WasSuccessful() bool {
	return j.Skipped == nil &&
		j.Error == nil &&
		j.Failure == nil
}

// WriteResultsToFileOrDie writes test results out to a file in xUnit format. Dies on any errors.
func WriteResultsToFileOrDie(graph *core.BuildGraph, filename string) {
	if err := os.MkdirAll(path.Dir(filename), core.DirPermissions); err != nil {
		log.Fatalf("Failed to create directory for test output")
	}
	xmlTestResults := jUnitXMLTestResults{}
	xmlTestResults.XMLName.Local = "testsuites"
	for _, target := range graph.AllTargets() {
		testSuite := target.Results
		if len(testSuite.TestCases) > 0 {
			xmlTestSuite := jUnitXMLTestSuite{
				Name:     target.Label.String(),
				Errors:   testSuite.Errors(),
				Failures: testSuite.Failures(),
				Skipped:  testSuite.Skips(),
				Tests:    testSuite.Tests(),
				Timed:    &Timed{testSuite.Duration().Seconds()},
				// TODO(agenticarus): Test groups not yet implemented, and don't tend to show up in UIs anyway.
				// Group:    "",
			}
			for _, testCase := range testSuite.TestCases {
				xmlTest := toXmlTestCase(testCase)
				xmlTestSuite.TestCases = append(xmlTestSuite.TestCases, xmlTest)
			}
			xmlTestResults.TestSuites = append(xmlTestResults.TestSuites, xmlTestSuite)
		}
	}
	if b, err := xml.MarshalIndent(xmlTestResults, "", "    "); err != nil {
		log.Fatalf("Failed to serialise XML: %s", err)
	} else if err = ioutil.WriteFile(filename, b, 0644); err != nil {
		log.Fatalf("Failed to write XML to %s: %s", filename, err)
	}
}

func toXmlTestCase(result core.TestCase) jUnitXMLTest {
	testcase := jUnitXMLTest{
		ClassName: result.ClassName,
		Name:      result.Name,
		// TODO(agenticarus): Test groups are not supported yet
		// Group: "",
	}
	success := result.Success()
	failures := result.Failures()
	errors := result.Errors()
	skip := result.Skip()
	if success != nil {
		// We passed but we might have had flakes
		testcase.Stderr = success.Stderr
		testcase.Stdout = success.Stdout
		testcase.Time = success.Duration.Seconds()
		for _, execution := range failures {
			testcase.FlakyFailure = append(testcase.FlakyFailure, jUnitXMLFlaky{
				Message:   execution.Failure.Message,
				Stderr:    execution.Stderr,
				Stdout:    execution.Stdout,
				Traceback: execution.Failure.Traceback,
				Type:      execution.Failure.Type,
			})
		}
		for _, execution := range errors {
			testcase.FlakyError = append(testcase.FlakyError, jUnitXMLFlaky{
				Message:   execution.Error.Message,
				Stderr:    execution.Stderr,
				Stdout:    execution.Stdout,
				Traceback: execution.Error.Traceback,
				Type:      execution.Error.Type,
			})
		}
	} else if skip != nil {
		testcase.Skipped = &jUnitXMLSkipped{
			Message: skip.Skip.Message,
		}
	} else {
		// We didn't have a single pass, everything is darkness
		// See if we 'failed' or 'errored' first.
		doneFirst := false
		setDuration := false
		for _, execution := range result.Executions {
			if execution.Error != nil {
				if !doneFirst {
					testcase.Error = &jUnitXMLError{
						Message:   execution.Error.Message,
						Traceback: execution.Error.Traceback,
						Type:      execution.Error.Type,
					}
					testcase.Stderr = execution.Stderr
					testcase.Stdout = execution.Stdout
					doneFirst = true
				} else {
					testcase.RerunError = append(testcase.RerunError, jUnitXMLRerunError{
						Message:   execution.Error.Message,
						Stderr:    execution.Stderr,
						Stdout:    execution.Stdout,
						Traceback: execution.Error.Traceback,
						Type:      execution.Error.Type,
					})
				}
			} else if execution.Failure != nil {
				if !doneFirst {
					testcase.Failure = &jUnitXMLFailure{
						Message:   execution.Error.Message,
						Traceback: execution.Error.Traceback,
						Type:      execution.Error.Type,
					}
					testcase.Stderr = execution.Stderr
					testcase.Stdout = execution.Stdout
					doneFirst = true
				} else {
					testcase.RerunFailure = append(testcase.RerunFailure, jUnitXMLRerunFailure{
						Message:   execution.Error.Message,
						Stderr:    execution.Stderr,
						Stdout:    execution.Stdout,
						Timed:     Timed{execution.Duration.Seconds()},
						Traceback: execution.Error.Traceback,
						Type:      execution.Error.Type,
					})
				}
				if !setDuration {
					testcase.Time = execution.Duration.Seconds()
					setDuration = true
				}
			}
		}
	}
	return testcase
}
