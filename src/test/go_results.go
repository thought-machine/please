// Parser for output from Go's testing package.
//
// This is a fairly straightforward microformat so pretty easy to parse ourselves.
// There's at least one package out there to convert it to JUnit XML but not worth
// the complexity of getting that installed as a standalone tool.

package test

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"core"
)

// Not sure what the -6 suffixes are about.
var testStart = regexp.MustCompile(`^=== RUN (.*)(?:-6)?$`)
var testResult = regexp.MustCompile(`^ *--- (PASS|FAIL|SKIP): (.*)(?:-6)? \(([0-9]+\.[0-9]+)s\)$`)

func parseGoTestResults(data []byte) (core.TestSuite, error) {
	results := core.TestSuite{}
	lines := bytes.Split(data, []byte{'\n'})
	testsStarted := map[string]bool{}
	var suiteDuration time.Duration
	for i, line := range lines {
		testStartMatches := testStart.FindSubmatch(line)
		testResultMatches := testResult.FindSubmatch(line)
		if testStartMatches != nil {
			testsStarted[strings.TrimSpace(string(testStartMatches[1]))] = true
		} else if testResultMatches != nil {
			testName := strings.TrimSpace(string(testResultMatches[2]))
			if !testsStarted[testName] {
				continue
			}
			f, _ := strconv.ParseFloat(string(testResultMatches[3]), 64)
			duration := time.Duration(f * float64(time.Second))
			suiteDuration += duration
			testCase := core.TestCase{
				Name: testName,
			}
			if bytes.Equal(testResultMatches[1], []byte("PASS")) {
				testCase.Executions = append(testCase.Executions, core.TestExecution{
					Duration: &duration,
				})
			} else if bytes.Equal(testResultMatches[1], []byte("SKIP")) {
				i++ // Following line has the reason for being skipped
				testCase.Executions = append(testCase.Executions, core.TestExecution{
					Skip: &core.TestResultSkip{
						Message: string(bytes.TrimSpace(lines[i])),
					},
					Duration: &duration,
				})
			} else {
				output := ""
				for j := i + 1; j < len(lines) && !bytes.HasPrefix(lines[j], []byte("===")); j++ {
					output += string(lines[j]) + "\n"
				}
				testCase.Executions = append(testCase.Executions, core.TestExecution{
					Failure: &core.TestResultFailure{
						Traceback: output,
					},
					Duration: &duration,
				})
			}
			results.TestCases = append(results.TestCases, testCase)
		} else if bytes.Equal(line, []byte("PASS")) {
			// Do nothing, all's well.
		} else if bytes.Equal(line, []byte("FAIL")) {
			if results.Failures() == 0 {
				return results, fmt.Errorf("Test indicated final failure but no failures found yet")
			}
		}
	}
	results.Duration = suiteDuration
	return results, nil
}
