// parser for output from Go's testing package.
//
// This is a fairly straightforward microformat so pretty easy to parse ourselves.
// There's at least one package out there to convert it to JUnit XML but not worth
// the complexity of getting that installed as a standalone tool.

package test

import (
	"bytes"
	"strings"

	"github.com/jstemmer/go-junit-report/v2/gtr"
	"github.com/jstemmer/go-junit-report/v2/parser/gotest"

	"github.com/thought-machine/please/src/core"
)

func parseGoTestResults(data []byte) (core.TestSuite, error) {
	parser := gotest.NewParser()
	report, err := parser.Parse(bytes.NewReader(data))
	if err != nil {
		return core.TestSuite{}, err
	}
	// We should only get a single package here; because of the way we set up tests, we only get results from one at a time.
	for _, pkg := range report.Packages {
		suite := core.TestSuite{
			Package:    pkg.Name,
			Duration:   pkg.Duration,
			Properties: pkg.Properties,
		}
		for _, test := range pkg.Tests {
			execution := core.TestExecution{
				Duration: &test.Duration,
			}
			output := strings.TrimSpace(strings.Join(test.Output, "\n"))
			switch test.Result {
			case gtr.Fail:
				execution.Failure = &core.TestResultFailure{
					Message: output,
				}
			case gtr.Skip:
				execution.Skip = &core.TestResultSkip{
					Message: output,
				}
			case gtr.Pass:
				execution.Stdout = output
			}
			suite.TestCases = append(suite.TestCases, core.TestCase{
				Name:       test.Name,
				Executions: []core.TestExecution{execution},
			})
		}
		return suite, nil
	}
	return core.TestSuite{}, nil
}
