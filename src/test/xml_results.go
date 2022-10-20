// parser for JUnit XML output.

package test

import (
	"bytes"
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/thought-machine/please/src/core"
)

var client = &http.Client{
	Timeout: time.Minute,
}

func looksLikeJUnitXMLTestResults(b []byte) bool {
	return bytes.HasPrefix(b, []byte{'<', '?', 'x', 'm', 'l'}) || bytes.HasPrefix(b, []byte{'<', 't', 'e', 's', 't'})
}

func parseJUnitXMLTestResults(data []byte) (core.TestSuites, error) {
	results := core.TestSuites{}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	for {
		token, err := decoder.Token()
		switch err {
		case nil:
		case io.EOF:
			return results, nil
		default:
			return results, err
		}

		switch tok := token.(type) { //nolint:gocritic
		case xml.StartElement:
			switch tok.Name.Local {
			case "test":
				// UnitTest.cpp Test
				uxmlTest := unitTestXMLTest{}
				decoder.DecodeElement(&uxmlTest, &tok)
				xmlTest := uxmlTest.toJUnitXMLTest()
				testSuite := core.TestSuite{
					Name: uxmlTest.Suite,
				}
				testCase := core.TestCase{
					Name: uxmlTest.Name,
				}
				appendResult(xmlTest, &testCase)
				testSuite.TestCases = append(testSuite.TestCases, testCase)
				testSuite.Duration += xmlTest.Duration()
				results.TestSuites = append(results.TestSuites, testSuite)
			case "testcase":
				// One or more bare tests, put each one in a synthetic test suite
				testSuite := core.TestSuite{}
				xmlTest := jUnitXMLTest{}
				testCase := core.TestCase{}
				decoder.DecodeElement(&xmlTest, &tok)
				appendResult(xmlTest, &testCase)
				testSuite.TestCases = append(testSuite.TestCases, testCase)
				testSuite.Duration += xmlTest.Duration()
				results.TestSuites = append(results.TestSuites, testSuite)
			case "testsuite": // Just a single test suite (this is the usual output from junit, for example)
				xmlTestSuite := &jUnitXMLTestSuite{}
				decoder.DecodeElement(xmlTestSuite, &tok)
				results.TestSuites = append(results.TestSuites, toCoreTestSuite(xmlTestSuite))
			case "testsuites": // We might have a collection of existing test suites, if we're parsing our own output.
				xmlTestSuites := jUnitXMLTestSuites{}
				decoder.DecodeElement(&xmlTestSuites, &tok)

				var duration time.Duration
				for _, xmlTestSuite := range xmlTestSuites.TestSuites {
					results.TestSuites = append(results.TestSuites, toCoreTestSuite(xmlTestSuite))
					duration += xmlTestSuite.Duration()
				}
			}
		}
	}
}

func toCoreTestSuite(xmlTestSuite *jUnitXMLTestSuite) core.TestSuite {
	testSuite := core.TestSuite{
		Package:    xmlTestSuite.Package,
		Name:       xmlTestSuite.Name,
		Timestamp:  xmlTestSuite.Timestamp,
		Duration:   xmlTestSuite.Duration(),
		Cached:     toCoreCached(xmlTestSuite.Properties),
		Properties: toCoreProperties(xmlTestSuite.Properties),
	}
	for _, test := range xmlTestSuite.TestCases {
		result := core.TestCase{
			ClassName: test.ClassName,
			Name:      test.Name,
		}
		appendResult(test, &result)
		testSuite.TestCases = append(testSuite.TestCases, result)
	}
	return testSuite
}

func toCoreCached(properties jUnitXMLProperties) bool {
	for _, prop := range properties.Property {
		if prop.Name == "cached" {
			if p, err := strconv.ParseBool(prop.Value); err == nil {
				return p
			}
			return false
		}
	}
	return false
}

func toCoreProperties(properties jUnitXMLProperties) map[string]string {
	props := make(map[string]string)
	for _, prop := range properties.Property {
		if prop.Name == "cached" {
			continue
		}
		props[prop.Name] = prop.Value
	}
	return props
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
	d := time.Duration(test.Time)
	results.Executions = append(results.Executions, core.TestExecution{
		Failure: &core.TestResultFailure{
			Message:   failure.Message,
			Type:      failure.Type,
			Traceback: failure.Traceback,
		},
		Duration: &d,
		Stdout:   test.Stdout,
		Stderr:   test.Stderr,
	})
}

func appendFlakyFailure(test jUnitXMLTest, results *core.TestCase, flake jUnitXMLFlaky) {
	d := time.Duration(test.Time)
	results.Executions = append(results.Executions, core.TestExecution{
		Failure: &core.TestResultFailure{
			Message:   flake.Message,
			Type:      flake.Type,
			Traceback: flake.Traceback,
		},
		Duration: &d,
		Stdout:   test.Stdout,
		Stderr:   test.Stderr,
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
	d := time.Duration(test.Time)
	results.Executions = append(results.Executions, core.TestExecution{
		Failure: &core.TestResultFailure{
			Message:   flake.Message,
			Type:      flake.Type,
			Traceback: flake.Traceback,
		},
		Duration: &d,
		Stdout:   test.Stdout,
		Stderr:   test.Stderr,
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

type jUnitXMLTestSuites struct {
	Errors   uint   `xml:"errors,attr,omitempty"`
	Failures uint   `xml:"failures,attr,omitempty"`
	Name     string `xml:"name,attr,omitempty"`
	Skipped  uint   `xml:"skipped,attr,omitempty"`
	Tests    uint   `xml:"tests,attr,omitempty"`
	timed    `xml:"time,attr,omitempty"`

	TestSuites []*jUnitXMLTestSuite `xml:"testsuite,omitempty"`

	XMLName xml.Name `xml:"testsuites"`
}

type jUnitXMLTestSuite struct {
	Name  string `xml:"name,attr"`
	Tests int    `xml:"tests,attr"`

	Errors    int    `xml:"errors,attr,omitempty"`
	Failures  int    `xml:"failures,attr,omitempty"`
	HostName  string `xml:"hostname,attr,omitempty"`
	Skipped   int    `xml:"skipped,attr,omitempty"`
	Package   string `xml:"package,attr,omitempty"`
	timed     `xml:"time,attr,omitempty"`
	Timestamp string `xml:"timestamp,attr,omitempty"`

	Properties jUnitXMLProperties `xml:"properties,omitempty"`
	TestCases  []jUnitXMLTest     `xml:"testcase"`
	Stdout     string             `xml:"system-out,omitempty"`
	Stderr     string             `xml:"system-err,omitempty"`

	XMLName xml.Name `xml:"testsuite"`
}

type jUnitXMLTest struct {
	Name string `xml:"name,attr"`

	Assertions uint   `xml:"assertions,attr,omitempty"`
	ClassName  string `xml:"classname,attr,omitempty"`
	Status     string `xml:"status,attr,omitempty"`
	timed      `xml:"time,attr,omitempty"`

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
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type jUnitXMLError struct {
	Message string `xml:"message,attr,omitempty"`
	Type    string `xml:"type,attr"`

	Traceback string `xml:",chardata"`
}

type jUnitXMLFailure struct {
	Message string `xml:"message,attr,omitempty"`
	Type    string `xml:"type,attr"`

	Traceback string `xml:",chardata"`
}

type jUnitXMLFlaky struct {
	Message string `xml:"message,attr,omitempty"`
	Type    string `xml:"type,attr"`

	Traceback string `xml:",chardata"`
	Stdout    string `xml:"system-out,omitempty"`
	Stderr    string `xml:"system-err,omitempty"`
}

type jUnitXMLRerunError struct {
	Message string `xml:"message,attr,omitempty"`
	Type    string `xml:"type,attr"`

	Traceback string `xml:",chardata"`
	Stdout    string `xml:"system-out,omitempty"`
	Stderr    string `xml:"system-err,omitempty"`
}

type jUnitXMLRerunFailure struct {
	Message string `xml:"message,attr,omitempty"`
	timed   `xml:"time,attr"`
	Type    string `xml:"type,attr"`

	Traceback string `xml:",chardata"`
	Stdout    string `xml:"system-out,omitempty"`
	Stderr    string `xml:"system-err,omitempty"`
}

type jUnitXMLSkipped struct {
	Message string `xml:"message,attr,omitempty"`
}

type timed struct {
	Time float64 `xml:"time,attr"`
}

func (t timed) Duration() time.Duration {
	return time.Duration(t.Time * float64(time.Second))
}

func (j jUnitXMLTest) WasSuccessful() bool {
	return j.Skipped == nil &&
		j.Error == nil &&
		j.Failure == nil
}

type unitTestXMLTest struct {
	Suite   string  `xml:"suite,attr"`
	Name    string  `xml:"name,attr"`
	Elapsed float64 `xml:"elapsed,attr"`

	Failure *unitTestXMLFailure `xml:"failure,omitempty"`
}

func (uxmlTest *unitTestXMLTest) toJUnitXMLTest() jUnitXMLTest {
	var failure *jUnitXMLFailure
	if uxmlTest.Failure != nil {
		failure = &jUnitXMLFailure{
			Message: uxmlTest.Failure.Message,
		}
	}
	return jUnitXMLTest{
		Name:      uxmlTest.Name,
		ClassName: uxmlTest.Suite,
		timed:     timed{uxmlTest.Elapsed},
		Failure:   failure,
	}
}

type unitTestXMLFailure struct {
	Message string `xml:"message,attr"`
}

// WriteResultsToFileOrDie writes test results out to a file in xUnit format. Dies on any errors.
func WriteResultsToFileOrDie(graph *core.BuildGraph, filename string, storeOutputOnSuccess bool) {
	if err := os.MkdirAll(filepath.Dir(filename), core.DirPermissions); err != nil {
		log.Fatalf("Failed to create directory for test output")
	} else if err = os.WriteFile(filename, mustSerialiseResults(graph, storeOutputOnSuccess), 0644); err != nil {
		log.Fatalf("Failed to write XML to %s: %s", filename, err)
	}
}

// SerialiseResultsToXML serialises some test results to the "standard" XML format.
func SerialiseResultsToXML(target *core.BuildTarget, indent, storeOutputOnSuccess bool) []byte {
	s := ""
	if indent {
		s = "    "
	}
	suite := toXMLTestSuite(target.Test.Results, storeOutputOnSuccess)
	suites := &jUnitXMLTestSuites{
		Name:       target.Label.String(),
		TestSuites: []*jUnitXMLTestSuite{suite},
	}
	suites.Time = suite.Time
	b, _ := xml.MarshalIndent(suites, "", s)
	return b
}

// uploadResults uploads test results to a remote server.
func uploadResults(target *core.BuildTarget, url string, gzipped, storeOutputOnSuccess bool) error {
	b := SerialiseResultsToXML(target, true, storeOutputOnSuccess)
	enc := ""
	var r io.Reader = bytes.NewReader(b)
	if gzipped {
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		if _, err := io.Copy(zw, r); err != nil {
			return fmt.Errorf("Failed to gzip test results: %s", err)
		} else if err = zw.Close(); err != nil {
			return fmt.Errorf("Failed to flush gzip writer: %s", err)
		}
		r = &buf
		enc = "gzip"
	}
	req, err := http.NewRequest(http.MethodPost, url, r)
	if err != nil {
		return fmt.Errorf("Failed to create HTTP request: %s", err)
	}
	req.Header["Content-Type"] = []string{"application/xml"}
	req.Header["Content-Encoding"] = []string{enc}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Failed to upload test results: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Error from remote server on uploading test results: %s", resp.Status)
	}
	return nil
}

// mustSerialiseResults serialises all test results into XML.
func mustSerialiseResults(graph *core.BuildGraph, storeOutputOnSuccess bool) []byte {
	xmlTestResults := jUnitXMLTestSuites{}
	xmlTestResults.XMLName.Local = "testsuites"

	// Collapse any testsuite with the same name
	xmlSuites := make(map[string]*jUnitXMLTestSuite)
	for _, target := range graph.AllTargets() {
		if target.IsTest() && target.Test.Results != nil {
			testSuite := target.Test.Results
			if len(testSuite.TestCases) > 0 {
				xmlTestSuite := toXMLTestSuite(testSuite, storeOutputOnSuccess)
				name := testSuite.JavaStyleName()
				if suite, present := xmlSuites[name]; present {
					suite.Tests += xmlTestSuite.Tests
					suite.Errors += xmlTestSuite.Errors
					suite.Failures += xmlTestSuite.Failures
					suite.Skipped += xmlTestSuite.Skipped
					suite.timed.Time += xmlTestSuite.timed.Time
					suite.TestCases = append(suite.TestCases, xmlTestSuite.TestCases...)
				} else {
					xmlSuites[name] = xmlTestSuite
				}
			}
			xmlTestResults.Time += testSuite.Duration.Seconds()
		}
	}
	for _, xmlTestSuite := range xmlSuites {
		xmlTestResults.TestSuites = append(xmlTestResults.TestSuites, xmlTestSuite)
	}
	b, err := xml.MarshalIndent(xmlTestResults, "", "    ")
	if err != nil {
		log.Fatalf("Failed to serialise XML: %s", err)
	}
	return b
}

func toXMLProperties(props map[string]string, cached bool) jUnitXMLProperties {
	out := jUnitXMLProperties{}
	for k, v := range props {
		out.Property = append(out.Property, jUnitXMLProperty{
			Name:  k,
			Value: v,
		})
	}
	if cached {
		out.Property = append(out.Property, jUnitXMLProperty{
			Name:  "cached",
			Value: strconv.FormatBool(cached),
		})
	}
	return out
}

func toXMLTestSuite(testSuite *core.TestSuite, storeOutputOnSuccess bool) *jUnitXMLTestSuite {
	xmlTestSuite := &jUnitXMLTestSuite{
		Name:       testSuite.Name,
		Package:    testSuite.Package,
		Timestamp:  testSuite.Timestamp,
		Tests:      testSuite.Tests(),
		Errors:     testSuite.Errors(),
		Failures:   testSuite.Failures(),
		Skipped:    testSuite.Skips(),
		timed:      timed{testSuite.Duration.Seconds()},
		Properties: toXMLProperties(testSuite.Properties, testSuite.Cached),
	}
	for _, testCase := range testSuite.TestCases {
		xmlTest := toXMLTestCase(testCase, storeOutputOnSuccess)
		if xmlTest.ClassName == "" {
			xmlTest.ClassName = testSuite.JavaStyleName()
		}
		xmlTestSuite.TestCases = append(xmlTestSuite.TestCases, xmlTest)
	}
	return xmlTestSuite
}

func toXMLTestCase(result core.TestCase, storeOutputOnSuccess bool) jUnitXMLTest {
	testcase := jUnitXMLTest{
		ClassName: result.ClassName,
		Name:      result.Name,
	}
	success := result.Success()
	failures := result.Failures()
	errors := result.Errors()
	skip := result.Skip()
	if success != nil {
		// We passed but we might have had flakes
		testcase.Time = success.Duration.Seconds()

		if storeOutputOnSuccess {
			testcase.Stderr = success.Stderr
			testcase.Stdout = success.Stdout
		}

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
						Message:   execution.Failure.Message,
						Traceback: execution.Failure.Traceback,
						Type:      execution.Failure.Type,
					}
					testcase.Stderr = execution.Stderr
					testcase.Stdout = execution.Stdout
					doneFirst = true
				} else {
					testcase.RerunFailure = append(testcase.RerunFailure, jUnitXMLRerunFailure{
						Message:   execution.Failure.Message,
						Stderr:    execution.Stderr,
						Stdout:    execution.Stdout,
						timed:     timed{execution.Duration.Seconds()},
						Traceback: execution.Failure.Traceback,
						Type:      execution.Failure.Type,
					})
				}
				if !setDuration && execution.Duration != nil {
					testcase.Time = execution.Duration.Seconds()
					setDuration = true
				}
			}
		}
	}
	return testcase
}
