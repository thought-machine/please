// parser for output from Go's testing package.
//
// This is a fairly straightforward microformat so pretty easy to parse ourselves.
// There's at least one package out there to convert it to JUnit XML but not worth
// the complexity of getting that installed as a standalone tool.

package test

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/peterebden/go-deferred-regex"

	"github.com/thought-machine/please/src/core"
)

// Not sure what the -6 suffixes are about.
var testStart = deferredregex.DeferredRegex{Re: `^=== RUN (.*)(?:-6)?$`}
var testResult = deferredregex.DeferredRegex{Re: `^ *--- (PASS|FAIL|SKIP): (.*)(?:-6)? \(([0-9]+\.[0-9]+)s\)$`}

func parseGoTestResults(data []byte) (core.TestSuite, error) {
	results := core.TestSuite{}
	lines := bytes.Split(data, []byte{'\n'})
	testsStarted := map[string]bool{}
	var suiteDuration time.Duration
	testOutput := make([]string, 0)
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
					Stderr:   strings.Join(testOutput, ""),
				})
			} else if bytes.Equal(testResultMatches[1], []byte("SKIP")) {
				// The skip message is found at the bottom of the test output segment.
				// Prior to Go 1.14 the test output segment follows the results line.
				// In Go 1.14 the test output segment sits between the start line and the results line.
				outputLines := getTestOutputLines(i, lines)
				skipMessage := ""
				if len(outputLines) > 0 {
					skipMessage = strings.TrimSpace(outputLines[len(outputLines)-1])
				}

				testCase.Executions = append(testCase.Executions, core.TestExecution{
					Skip: &core.TestResultSkip{
						Message: skipMessage,
					},
					Stderr:   strings.Join(testOutput, ""),
					Duration: &duration,
				})
			} else {
				outputLines := getTestOutputLines(i, lines)

				output := strings.Join(outputLines, "\n")
				testCase.Executions = append(testCase.Executions, core.TestExecution{
					Failure: &core.TestResultFailure{
						Traceback: output,
					},
					Stderr:   strings.Join(testOutput, ""),
					Duration: &duration,
				})
			}
			results.TestCases = append(results.TestCases, testCase)
			testOutput = make([]string, 0)
		} else if bytes.Equal(line, []byte("PASS")) {
			// Do nothing, all's well.
		} else if bytes.Equal(line, []byte("FAIL")) {
			if results.Failures() == 0 {
				return results, fmt.Errorf("Test indicated final failure but no failures found yet")
			}
		} else {
			testOutput = append(testOutput, string(line), "\n")
		}
	}
	results.Duration = suiteDuration
	return results, nil
}

func getTestOutputLines(currentIndex int, lines [][]byte) []string {
	if resultLooksPriorGo114(currentIndex, lines) {
		return getPostResultOutput(currentIndex, lines)
	}
	return append(getPreResultOutput(currentIndex, lines), getPostResultOutput(currentIndex, lines)...)
}

// Go test output looks prior to 114 if the previous line matches against a start test block.
// Only fully applicable for failing and skipped tests as a message may not
// appear for passed tests.
func resultLooksPriorGo114(currentIndex int, lines [][]byte) bool {
	if currentIndex == 0 {
		return false
	}

	prevLine := lines[currentIndex-1]
	prevLineMatchesStart := testStart.FindSubmatch(prevLine)

	return prevLineMatchesStart != nil
}

// Get the output for Go test output prior to Go 1.14
func getPostResultOutput(resultsIndex int, lines [][]byte) []string {
	output := []string{}
	for j := resultsIndex + 1; j < len(lines) && !lineMatchesRunOrResultsLine(lines[j]); j++ {
		output = append(output, string(lines[j]))
	}

	return output
}

// Get output for Go tests output after Go 1.14
func getPreResultOutput(resultsIndex int, lines [][]byte) []string {
	output := []string{}
	for j := resultsIndex - 1; j > 0 && !lineMatchesRunOrResultsLine(lines[j]); j-- {
		output = append([]string{string(lines[j])}, output...)
	}
	return output
}

func lineMatchesRunOrResultsLine(line []byte) bool {
	testStartMatches := testStart.FindSubmatch(line)
	matchesRunLine := (testStartMatches != nil)

	return matchesRunLine || bytes.Equal(line, []byte("PASS")) || bytes.Equal(line, []byte("FAIL"))
}
