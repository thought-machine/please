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

func propertiesToMap(ps []gtr.Property) map[string]string {
	ret := make(map[string]string, len(ps))
	for _, p := range ps {
		ret[p.Name] = p.Value
	}
	return ret
}

func parseGoTestResults(data []byte) (core.TestSuite, error) {
	parser := gotest.NewParser()
	report, err := parser.Parse(bytes.NewReader(data))
	if err != nil {
		return core.TestSuite{}, err
	}
	if len(report.Packages) == 0 {
		return core.TestSuite{}, nil
	}
	// We should only get a single package here; because of the way we set up tests, we only get results from one at a time.
	pkg := report.Packages[0]
	suite := core.TestSuite{
		Package:    pkg.Name,
		Duration:   pkg.Duration,
		Properties: propertiesToMap(pkg.Properties),
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
	// Attach any trailing output to the last failing test case. This happens if e.g. a case panics.
	if len(suite.TestCases) > 0 {
		if c := &suite.TestCases[len(suite.TestCases)-1]; c.Executions[0].Failure != nil {
			c.Executions[0].Failure.Traceback = strings.Join(pkg.Output, "\n")
		}
	}
	return suite, nil
}
