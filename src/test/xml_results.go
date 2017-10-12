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
	junitCase := jUnitXMLTestResults{}
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

func appendResult(test jUnitXMLTest, results *core.TestResults) {
	results.NumTests++
	if test.Failure != nil {
		appendResult2(test, results, *test.Failure)
	} else if test.Error != nil {
		appendResult2(test, results, *test.Error)
	} else if test.Type == "FAILURE" || test.Success == "false" || test.Stacktrace != "" {
		appendResult2(test, results, jUnitXMLFailure{"", "FAILURE", test.Stacktrace})
	} else {
		results.Passed++
		results.Passes = append(results.Passes, test.Name)
	}
}

func appendResult2(test jUnitXMLTest, results *core.TestResults, failure jUnitXMLFailure) {
	results.Failed++
	results.Failures = append(results.Failures, core.TestFailure{
		Name:      combineNames(test.ClassName, test.Name),
		Type:      failure.Type,
		Traceback: messageOrTraceback(failure), // TODO(pebers): store both of these, not just one.
		Stdout:    test.Stdout,
		Stderr:    test.Stderr,
	})
}

func messageOrTraceback(failure jUnitXMLFailure) string {
	if failure.Traceback != "" {
		return failure.Traceback
	}
	return failure.Message
}

func combineNames(className string, name string) string {
	index := strings.LastIndex(className, ".")
	if index != -1 {
		return className[index+1:] + "." + name
	}
	return className + "." + name
}

type jUnitXMLTestResults struct {
	TestSuites []jUnitXMLTestSuite `xml:"testsuite"`
	TestCases  []jUnitXMLTest      `xml:"testcase"`
	Tests      []jUnitXMLTest      `xml:"test"`
	XMLName    xml.Name
}

type jUnitXMLTestSuite struct {
	Name      string         `xml:"name,attr"`
	Failures  int            `xml:"failures,attr,omitempty"`
	Tests     int            `xml:"tests,attr"`
	TestCases []jUnitXMLTest `xml:"testcase"`
}

type jUnitXMLTest struct {
	ClassName  string           `xml:"classname,attr,omitempty"`
	Name       string           `xml:"name,attr"`
	Failure    *jUnitXMLFailure `xml:"failure,omitempty"`
	Error      *jUnitXMLFailure `xml:"error,omitempty"`
	Time       float64          `xml:"time,attr,omitempty"`
	Type       string           `xml:"type,attr,omitempty"`
	Success    string           `xml:"success,attr,omitempty"`
	Stacktrace string           `xml:"stacktrace,attr,omitempty"`
	Stdout     string           `xml:"stdout,omitempty"`
	Stderr     string           `xml:"stderr,omitempty"`
}

type jUnitXMLFailure struct {
	Message   string `xml:"message,attr,omitempty"`
	Type      string `xml:"type,attr,omitempty"`
	Traceback string `xml:",chardata"`
}

// WriteResultsToFileOrDie writes test results out to a file in xUnit format. Dies on any errors.
func WriteResultsToFileOrDie(graph *core.BuildGraph, filename string) {
	if err := os.MkdirAll(path.Dir(filename), core.DirPermissions); err != nil {
		log.Fatalf("Failed to create directory for test output")
	}
	results := jUnitXMLTestResults{}
	results.XMLName.Local = "testsuites"
	for _, target := range graph.AllTargets() {
		if target.Results.NumTests > 0 {
			suite := jUnitXMLTestSuite{
				Name:     target.Label.String(),
				Failures: target.Results.Failed,
				Tests:    target.Results.NumTests,
			}
			for _, pass := range target.Results.Passes {
				suite.TestCases = append(suite.TestCases, jUnitXMLTest{Name: pass})
			}
			for _, fail := range target.Results.Failures {
				suite.TestCases = append(suite.TestCases, jUnitXMLTest{
					Name:   fail.Name,
					Type:   fail.Type,
					Stdout: fail.Stdout,
					Stderr: fail.Stderr,
					Error: &jUnitXMLFailure{
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
