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

func parseGoTestResults(data []byte) (core.TestResults, error) {
	results := core.TestResults{}
	lines := bytes.Split(data, []byte{'\n'})
	testsStarted := map[string]bool{}
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
			results.NumTests++
			if bytes.Equal(testResultMatches[1], []byte("PASS")) {
				results.Passed++
				results.Results = append(results.Results, core.TestResult{
					Name: testName, Success: true, Duration: duration,
				})
			} else if bytes.Equal(testResultMatches[1], []byte("SKIP")) {
				results.Skipped++
				i++ // Following line has the reason for being skipped
				results.Results = append(results.Results, core.TestResult{
					Name: testName, Skipped: true, Duration: duration, Type: string(bytes.TrimSpace(lines[i])),
				})
			} else {
				output := ""
				for j := i + 1; j < len(lines) && !bytes.HasPrefix(lines[j], []byte("===")); j++ {
					output += string(lines[j]) + "\n"
				}
				results.Failed++
				results.Results = append(results.Results, core.TestResult{
					Name: testName, Type: "FAILURE", Traceback: output, Stdout: "", Stderr: "", Duration: duration,
				})
			}
		} else if bytes.Equal(line, []byte("PASS")) {
			// Do nothing, all's well.
		} else if bytes.Equal(line, []byte("FAIL")) {
			if results.Failed == 0 {
				return results, fmt.Errorf("Test indicated final failure but no failures found yet")
			}
		}
	}
	return results, nil
}
