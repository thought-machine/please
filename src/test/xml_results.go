// Parser for JUnit XML output.

package test

import (
	"encoding/xml"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"core"
)

func parseJUnitXMLTestResults(bytes []byte) (core.TestResults, error) {
	results := core.TestResults{}
	junitCase := JUnitXMLTestResults{}
	if err := xml.Unmarshal(bytes, &junitCase); err != nil {
		return results, err
	}
	for _, test := range junitCase.Tests {
		appendResult(test, &results)
	}
	for _, test := range junitCase.TestCases {
		appendResult(test, &results)
	}
	for _, suite := range junitCase.TestSuites {
		for _, test := range suite.TestCases {
			appendResult(test, &results)
		}
	}
	return results, nil
}

func appendResult(test JUnitXMLTest, results *core.TestResults) {
	results.NumTests++
	if test.Failure != nil {
		appendResult2(test, results, *test.Failure)
	} else if test.Error != nil {
		appendResult2(test, results, *test.Error)
	} else if test.Type == "FAILURE" || test.Success == "false" || test.Stacktrace != "" {
		appendResult2(test, results, JUnitXMLFailure{"", "FAILURE", test.Stacktrace})
	} else {
		results.Passed++
		results.Passes = append(results.Passes, test.Name)
	}
}

func appendResult2(test JUnitXMLTest, results *core.TestResults, failure JUnitXMLFailure) {
	results.Failed++
	results.Failures = append(results.Failures, core.TestFailure{
		Name:      combineNames(test.ClassName, test.Name),
		Type:      failure.Type,
		Traceback: messageOrTraceback(failure), // TODO(pebers): store both of these, not just one.
		Stdout:    test.Stdout,
		Stderr:    test.Stderr,
	})
}

func messageOrTraceback(failure JUnitXMLFailure) string {
	if failure.Traceback != "" {
		return failure.Traceback
	} else {
		return failure.Message
	}
}

func combineNames(className string, name string) string {
	index := strings.LastIndex(className, ".")
	if index != -1 {
		return className[index+1:] + "." + name
	} else {
		return className + "." + name
	}
}

type JUnitXMLTestResults struct {
	TestSuites []JUnitXMLTestSuite `xml:"testsuite"`
	TestCases  []JUnitXMLTest      `xml:"testcase"`
	Tests      []JUnitXMLTest      `xml:"test"`
	XMLName    xml.Name
}

type JUnitXMLTestSuite struct {
	Name      string         `xml:"name,attr"`
	Failures  int            `xml:"failures,attr,omitempty"`
	Tests     int            `xml:"tests,attr"`
	TestCases []JUnitXMLTest `xml:"testcase"`
}

type JUnitXMLTest struct {
	ClassName  string           `xml:"classname,attr,omitempty"`
	Name       string           `xml:"name,attr"`
	Failure    *JUnitXMLFailure `xml:"failure,omitempty"`
	Error      *JUnitXMLFailure `xml:"error,omitempty"`
	Time       float64          `xml:"time,attr,omitempty"`
	Type       string           `xml:"type,attr,omitempty"`
	Success    string           `xml:"success,attr,omitempty"`
	Stacktrace string           `xml:"stacktrace,attr,omitempty"`
	Stdout     string           `xml:"stdout,omitempty"`
	Stderr     string           `xml:"stderr,omitempty"`
}

type JUnitXMLFailure struct {
	Message   string `xml:"message,attr,omitempty"`
	Type      string `xml:"type,attr,omitempty"`
	Traceback string `xml:",chardata"`
}

// Write test results out to a file in xUnit format. Dies on any errors.
func WriteResultsToFileOrDie(graph *core.BuildGraph, filename string) {
	if err := os.MkdirAll(path.Dir(filename), core.DirPermissions); err != nil {
		log.Fatalf("Failed to create directory for test output")
	}
	results := JUnitXMLTestResults{}
	results.XMLName.Local = "testsuites"
	for _, target := range graph.AllTargets() {
		if target.Results.NumTests > 0 {
			suite := JUnitXMLTestSuite{
				Name:     target.Label.String(),
				Failures: target.Results.Failed,
				Tests:    target.Results.NumTests,
			}
			for _, pass := range target.Results.Passes {
				suite.TestCases = append(suite.TestCases, JUnitXMLTest{Name: pass})
			}
			for _, fail := range target.Results.Failures {
				suite.TestCases = append(suite.TestCases, JUnitXMLTest{
					Name:   fail.Name,
					Type:   fail.Type,
					Stdout: fail.Stdout,
					Stderr: fail.Stderr,
					Error: &JUnitXMLFailure{
						Type:      fail.Type,
						Traceback: fail.Traceback,
					},
				})
			}
			results.TestSuites = append(results.TestSuites, suite)
		}
	}
	if b, err := xml.MarshalIndent(results, "", "    "); err != nil {
		log.Fatalf("Failed to serialise XML: %s", err)
	} else if err = ioutil.WriteFile(filename, b, 0644); err != nil {
		log.Fatalf("Failed to write XML to %s: %s", filename, err)
	}
}
