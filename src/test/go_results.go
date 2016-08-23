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
	"strings"

	"core"
)

func looksLikeGoTestResults(results []byte) bool {
	// The latter case happens when there are no test functions to run.
	return bytes.HasPrefix(results, []byte("===")) || bytes.HasPrefix(results, []byte("PASS"))
}

// Not sure what the -6 suffixes are about.
var testStart = regexp.MustCompile("^=== RUN (.*)(?:-6)?$")
var testResult = regexp.MustCompile("^ *--- (PASS|FAIL|SKIP): (.*)(?:-6)? \\((.*)s\\)$")

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
			results.NumTests++
			if bytes.Equal(testResultMatches[1], []byte("PASS")) {
				results.Passed++
				results.Passes = append(results.Passes, testName)
			} else if bytes.Equal(testResultMatches[1], []byte("SKIP")) {
				results.Skipped++
				i++ // Skip following line too that has the reason for being skipped
			} else {
				output := ""
				for j := i + 1; j < len(lines) && !bytes.HasPrefix(lines[j], []byte("===")); j++ {
					output += string(lines[j]) + "\n"
					i = j
				}
				results.Failed++
				results.Failures = append(results.Failures, core.TestFailure{
					Name: testName, Type: "FAILURE", Traceback: output, Stdout: "", Stderr: "",
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
