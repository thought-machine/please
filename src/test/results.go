// Code for parsing the output of tests.

package test

import "fmt"
import "io/ioutil"
import "os"
import "path/filepath"

import "core"

func parseTestResults(target *core.BuildTarget, outputFile string, flakes int, cached bool) (core.TestResults, error) {
	results, err := parseTestResultsDir(target, outputFile)
	results.Flakes = flakes
	results.Cached = cached
	target.Results.Aggregate(results)
	// Ensure that the target has a failure if we encountered an error
	if err != nil && target.Results.Failed == 0 {
		target.Results.NumTests++
		target.Results.Failed++
	}
	return results, err
}

func parseTestResultsImpl(target *core.BuildTarget, outputFile string) (core.TestResults, error) {
	bytes, err := ioutil.ReadFile(outputFile)
	if err != nil {
		return core.TestResults{}, err
	}
	if len(bytes) == 0 {
		return core.TestResults{}, fmt.Errorf("No results")
	} else if looksLikeGoTestResults(bytes) {
		return parseGoTestResults(bytes)
	} else {
		return parseJUnitXMLTestResults(bytes)
	}
}

func parseTestResultsDir(target *core.BuildTarget, outputDir string) (core.TestResults, error) {
	results := core.TestResults{}
	err := filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		} else if !info.IsDir() {
			fileResults, err := parseTestResultsImpl(target, path)
			if err != nil {
				return fmt.Errorf("Error parsing %s: %s", path, err)
			}
			results.Aggregate(fileResults)
		}
		return nil
	})
	if err == nil && results.NumTests == 0 {
		return results, fmt.Errorf("Didn't find any test results in %s", outputDir)
	}
	return results, err
}
