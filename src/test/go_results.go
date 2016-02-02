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
	return bytes.HasPrefix(results, []byte("==="))
}

// Not sure what the -6 suffixes are about.
var testStart = regexp.MustCompile("^=== RUN (.*)(?:-6)?$")
var testResult = regexp.MustCompile("^--- (PASS|FAIL|SKIP): (.*)(?:-6)? \\((.*)s\\)$")

func parseGoTestResults(data []byte) (core.TestResults, error) {
	results := core.TestResults{}
	lines := bytes.Split(data, []byte{'\n'})
	lastTestStarted := ""
	for i := 0; i < len(lines); i++ {
		testStartMatches := testStart.FindSubmatch(lines[i])
		testResultMatches := testResult.FindSubmatch(lines[i])
		if testStartMatches != nil {
			lastTestStarted = strings.TrimSpace(string(testStartMatches[1]))
		} else if testResultMatches != nil {
			testName := strings.TrimSpace(string(testResultMatches[2]))
			if testName != lastTestStarted {
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
		} else if bytes.Equal(lines[i], []byte("PASS")) {
			// Do nothing, all's well.
		} else if bytes.Equal(lines[i], []byte("FAIL")) {
			if results.Failed == 0 {
				return results, fmt.Errorf("Test indicated final failure but no failures found yet")
			}
		}
	}
	return results, nil
}
