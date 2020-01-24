// Code for parsing the output of tests.

package test

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

func parseTestResults(outputFile string, data [][]byte) (core.TestSuite, error) {
	if len(data) == 0 {
		return parseTestResultsDir(outputFile)
	}
	suite := core.TestSuite{}
	for _, datum := range data {
		newSuite, err := parseTestResultData(datum)
		if err != nil {
			return suite, err
		}
		suite.Collapse(newSuite)
	}
	return suite, nil
}

func parseTestResultsFile(outputFile string) (core.TestSuite, error) {
	data, err := ioutil.ReadFile(outputFile)
	if err != nil {
		return core.TestSuite{}, err
	}
	return parseTestResultData(data)
}

func parseTestResultData(data []byte) (core.TestSuite, error) {
	if len(data) == 0 {
		return core.TestSuite{}, fmt.Errorf("No results")
	} else if looksLikeJUnitXMLTestResults(data) {
		testSuites, err := parseJUnitXMLTestResults(data)
		testSuite := core.TestSuite{}
		for _, suite := range testSuites.TestSuites {
			testSuite.Collapse(suite)
		}
		return testSuite, err
	} else {
		return parseGoTestResults(data)
	}
}

func parseTestResultsDir(outputDir string) (core.TestSuite, error) {
	results := core.TestSuite{}
	if !core.PathExists(outputDir) {
		return results, fmt.Errorf("Didn't find any test results in %s", outputDir)
	}
	err := fs.Walk(outputDir, func(path string, isDir bool) error {
		if !isDir {
			fileResults, err := parseTestResultsFile(path)
			if err != nil {
				return fmt.Errorf("Error parsing %s: %s", path, err)
			}
			results.Collapse(fileResults)
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
	junit := jUnitXMLTestSuites{}
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
