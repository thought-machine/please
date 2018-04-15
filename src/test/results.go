// Code for parsing the output of tests.

package test

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"

	"core"
	"fs"
)

func parseTestResults(target *core.BuildTarget, outputFile string, cached bool) (core.TestResults, error) {
	results, err := parseTestResultsDir(outputFile)
	results.Cached = cached
	target.Results.Aggregate(&results)
	// Ensure that the target has a failure if we encountered an error
	if err != nil && target.Results.Failed == 0 {
		target.Results.NumTests++
		target.Results.Failed++
	}
	// Ensure that there is one success if the target succeeded but there are no tests.
	if err == nil && target.Results.Failed == 0 && target.Results.NumTests == 0 {
		target.Results.NumTests++
		target.Results.Passed++
	}
	return results, err
}

func parseTestResultsImpl(outputFile string) (core.TestResults, error) {
	bytes, err := ioutil.ReadFile(outputFile)
	if err != nil {
		return core.TestResults{}, err
	}
	if len(bytes) == 0 {
		return core.TestResults{}, fmt.Errorf("No results")
	} else if looksLikeJUnitXMLTestResults(bytes) {
		return parseJUnitXMLTestResults(bytes)
	} else {
		return parseGoTestResults(bytes)
	}
}

func parseTestResultsDir(outputDir string) (core.TestResults, error) {
	results := core.TestResults{}
	if !core.PathExists(outputDir) {
		return results, fmt.Errorf("Didn't find any test results in %s", outputDir)
	}
	err := fs.Walk(outputDir, func(path string, isDir bool) error {
		if !isDir {
			fileResults, err := parseTestResultsImpl(path)
			if err != nil {
				return fmt.Errorf("Error parsing %s: %s", path, err)
			}
			results.Aggregate(&fileResults)
		}
		return nil
	})
	return results, err
}

// LoadPreviousFailures loads any failed tests from the given results file.
// It returns the set of targets that should be run and any arguments for them.
func LoadPreviousFailures(filename string) ([]core.BuildLabel, []string) {
	f, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Failed to read previous test results: %s", err)
	}
	defer f.Close()
	// We have to read directly since the TestResults struct doesn't have all the information
	// we'll need (e.g. it discards test suite names).
	junit := jUnitXMLTestResults{}
	if err := xml.NewDecoder(f).Decode(&junit); err != nil {
		log.Fatalf("Failed to read previous test results: %s", err)
	}
	labels := []core.BuildLabel{}
	args := []string{}
	for _, suite := range junit.TestSuites {
		if suite.Failures > 0 {
			labels = append(labels, core.ParseBuildLabel(suite.Name, "")) // These always have complete labels
			for _, c := range suite.TestCases {
				if c.Failure != nil || c.Error != nil {
					args = append(args, c.Name)
				}
			}
		}
	}
	return labels, args
}
