// Parser for output from Go's testing package.

package test

import (
	"bytes"
	"fmt"
	"strings"

	parser "github.com/jstemmer/go-junit-report/parser"

	"github.com/thought-machine/please/src/core"
)

func parseGoTestResults(data []byte) (core.TestSuite, error) {

	testPackage, err := parser.Parse(bytes.NewReader(data), "")

	if err != nil {
		return core.TestSuite{}, fmt.Errorf("Failed to parser go test output: %w", err)
	}

	results := fromGoJunitReport(testPackage)

	return results, nil
}

// Conversion between a go-junit-report Report into a core.TestSuite.
// A Package is mapped to TestSuite & Tests mapped onto testCases.
func fromGoJunitReport(report *parser.Report) core.TestSuite {
	results := core.TestSuite{}

	for _, pkg := range report.Packages {
		for _, test := range pkg.Tests {
			coreTestCase := core.TestCase{Name: test.Name}
			testOutput := strings.Join(test.Output, "\n")

			if test.Result == parser.PASS {
				coreTestCase.Executions = append(coreTestCase.Executions, core.TestExecution{
					Stderr:   testOutput,
					Duration: &test.Duration,
				})
			} else if test.Result == parser.SKIP {
				coreTestCase.Executions = append(coreTestCase.Executions, core.TestExecution{
					Skip: &core.TestResultSkip{
						// Given the possibility of test setup, teardowns & custom logging, we can't do anything
						// more targeted than using the whole test output as the skip message.
						Message: testOutput,
					},
					Stderr:   testOutput,
					Duration: &test.Duration,
				})
			} else {
				// A "FAIL" result
				coreTestCase.Executions = append(coreTestCase.Executions, core.TestExecution{
					Failure: &core.TestResultFailure{
						Traceback: testOutput,
					},
					Stderr:   testOutput,
					Duration: &test.Duration,
				})
			}
			results.TestCases = append(results.TestCases, coreTestCase)
		}
		results.Duration += pkg.Duration
	}
	return results
}
