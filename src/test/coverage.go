// Code for parsing coverage output in various formats.

package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
)

// Parses test coverage for a single target from its output file.
func parseTestCoverage(target *core.BuildTarget, outputFile string) (core.TestCoverage, error) {
	coverage := core.NewTestCoverage()
	data, err := ioutil.ReadFile(outputFile)
	if err != nil && os.IsNotExist(err) {
		return coverage, nil // Tests aren't required to produce coverage files.
	} else if err != nil {
		return coverage, err
	} else if len(data) == 0 {
		return coverage, fmt.Errorf("Empty coverage output")
	} else if looksLikeGoCoverageResults(data) {
		// TODO(pebers): this is a little wasteful, we've already read the file once and we must do it again.
		return coverage, parseGoCoverageResults(target, &coverage, outputFile)
	} else if looksLikeGcovCoverageResults(data) {
		return coverage, parseGcovCoverageResults(target, &coverage, data)
	} else if looksLikeIstanbulCoverageResults(data) {
		return coverage, parseIstanbulCoverageResults(target, &coverage, data)
	} else {
		return coverage, parseXMLCoverageResults(target, &coverage, data)
	}
}

// AddOriginalTargetsToCoverage adds empty coverage entries for any files covered by the original
// query that we haven't discovered through tests to the overall report.
// The coverage reports only contain information about files that were covered during
// tests, so it's important that we identify anything with zero coverage here.
// This is made trickier by attempting to reconcile coverage targets from languages like
// Java that don't preserve the original file structure, which requires a slightly fuzzy match.
func AddOriginalTargetsToCoverage(state *core.BuildState, includeAllFiles bool) {
	// First we collect all the source files from all relevant targets
	allFiles := map[string]bool{}
	doneTargets := map[*core.BuildTarget]bool{}
	// Track the set of packages the user ran tests from; we only show coverage metrics from them.
	coveragePackages := map[string]bool{}
	for _, label := range state.OriginalTargets {
		coveragePackages[label.PackageName] = true
	}
	for _, label := range state.ExpandOriginalTargets() {
		collectAllFiles(state, state.Graph.TargetOrDie(label), coveragePackages, allFiles, doneTargets, includeAllFiles)
	}

	// Now merge the recorded coverage so far into them
	recordedCoverage := state.Coverage
	state.Coverage = core.TestCoverage{Tests: recordedCoverage.Tests, Files: map[string][]core.LineCoverage{}}
	mergeCoverage(state, recordedCoverage, coveragePackages, allFiles, includeAllFiles)
}

// Collects all the source files from a single target
func collectAllFiles(state *core.BuildState, target *core.BuildTarget, coveragePackages, allFiles map[string]bool, doneTargets map[*core.BuildTarget]bool, includeAllFiles bool) {
	doneTargets[target] = true
	if !includeAllFiles && !coveragePackages[target.Label.PackageName] {
		return
	}
	// Small hack here; explore these targets when we don't have any sources yet. Helps languages
	// like Java where we generate a wrapper target with a complete one immediately underneath.
	// TODO(pebers): do we still need this now we have Java sourcemaps?
	if !target.OutputIsComplete || len(allFiles) == 0 {
		for _, dep := range target.Dependencies() {
			if !doneTargets[dep] {
				collectAllFiles(state, dep, coveragePackages, allFiles, doneTargets, includeAllFiles)
			}
		}
	}
	if target.IsTest {
		return // Test sources don't count for coverage.
	}
	for _, path := range target.AllSourcePaths(state.Graph) {
		extension := filepath.Ext(path)
		for _, ext := range state.Config.Cover.FileExtension {
			if ext == extension {
				allFiles[path] = target.IsTest || target.TestOnly // Skip test source files from actual coverage display
				break
			}
		}
	}
}

// mergeCoverage merges recorded coverage with the list of all existing files.
func mergeCoverage(state *core.BuildState, recordedCoverage core.TestCoverage, coveragePackages, allFiles map[string]bool, includeAllFiles bool) {
	for file, coverage := range recordedCoverage.Files {
		if includeAllFiles || isOwnedBy(file, coveragePackages) {
			state.Coverage.Files[file] = coverage
			allFiles[file] = true
		}
	}
	// For any files left over now, enter them in as 100% uncovered.
	// This is pessimistic but there's not much we can do at this point.
	for file, done := range allFiles {
		if !done {
			s := make([]core.LineCoverage, countLines(file))
			if len(s) > 0 {
				for i := 0; i < len(s); i++ {
					s[i] = core.Uncovered
				}
				state.Coverage.Files[file] = s
			}
		}
	}
}

// isOwnedBy returns true if the given file is owned by any of the given packages.
func isOwnedBy(file string, coveragePackages map[string]bool) bool {
	for file != "." && file != "/" {
		file = path.Dir(file)
		if coveragePackages[file] {
			return true
		}
	}
	return false
}

// countLines returns the number of lines in a file.
func countLines(path string) int {
	data, _ := ioutil.ReadFile(path)
	return bytes.Count(data, []byte{'\n'})
}

// WriteCoverageToFileOrDie writes the collected coverage data to a file in JSON format. Dies on failure.
func WriteCoverageToFileOrDie(coverage core.TestCoverage, filename string) {
	out := jsonCoverage{Tests: map[string]map[string]string{}}
	allowedFiles := coverage.OrderedFiles()

	for label, coverage := range coverage.Tests {
		out.Tests[label.String()] = convertCoverage(coverage, allowedFiles)
	}

	out.Files = convertCoverage(coverage.Files, allowedFiles)
	out.Stats = getStats(coverage)
	if b, err := json.MarshalIndent(out, "", "    "); err != nil {
		log.Fatalf("Failed to encode json: %s", err)
	} else if err := ioutil.WriteFile(filename, b, 0644); err != nil {
		log.Fatalf("Failed to write coverage results to %s: %s", filename, err)
	}
}

// WriteXMLCoverageToFileOrDie writes the collected coverage data to a file in XML format. Dies on failure.
func WriteXMLCoverageToFileOrDie(sources []core.BuildLabel, coverage core.TestCoverage, filename string) {
	data := coverageResultToXML(sources, coverage)

	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		log.Fatalf("Failed to write coverage results to %s: %s", filename, err)
	}
}

// CountCoverage counts the number of lines covered and the total number coverable in a single file.
func CountCoverage(lines []core.LineCoverage) (int, int) {
	covered := 0
	total := 0
	for _, line := range lines {
		if line == core.Covered {
			total++
			covered++
		} else if line != core.NotExecutable {
			total++
		}
	}
	return covered, total
}

func getStats(coverage core.TestCoverage) stats {
	stats := stats{CoverageByFile: map[string]float32{}}
	totalLinesCovered := 0
	totalCoverableLines := 0
	for _, file := range coverage.OrderedFiles() {
		covered, total := CountCoverage(coverage.Files[file])
		totalLinesCovered += covered
		totalCoverableLines += total
		if total > 0 {
			stats.CoverageByFile[file] = 100.0 * float32(covered) / float32(total)
		}
	}
	if totalCoverableLines > 0 {
		stats.TotalCoverage = 100.0 * float32(totalLinesCovered) / float32(totalCoverableLines)
	}
	return stats
}

func convertCoverage(in map[string][]core.LineCoverage, allowedFiles []string) map[string]string {
	ret := map[string]string{}
	for k, v := range in {
		if cli.ContainsString(k, allowedFiles) {
			ret[k] = core.TestCoverageString(v)
		}
	}
	return ret
}

// Used to prepare core.TestCoverage objects for JSON marshalling.
type jsonCoverage struct {
	Tests map[string]map[string]string `json:"tests"`
	Files map[string]string            `json:"files"`
	Stats stats                        `json:"stats"`
}

// stats is a struct describing summarised coverage stats.
type stats struct {
	TotalCoverage  float32            `json:"total_coverage"`
	CoverageByFile map[string]float32 `json:"coverage_by_file"`
}

// RemoveFilesFromCoverage removes any files with extensions matching the given set from coverage.
func RemoveFilesFromCoverage(coverage core.TestCoverage, extensions []string) {
	for _, files := range coverage.Tests {
		removeFilesFromCoverage(files, extensions)
	}
	removeFilesFromCoverage(coverage.Files, extensions)
}

func removeFilesFromCoverage(files map[string][]core.LineCoverage, extensions []string) {
	for filename := range files {
		for _, ext := range extensions {
			if strings.HasSuffix(filename, ext) {
				delete(files, filename)
			}
		}
	}
}
