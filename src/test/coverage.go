// Code for parsing coverage output in various formats.

package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

// Parses test coverage for a single target from its output file.
func parseTestCoverageFile(target *core.BuildTarget, outputFile string, run int) (*core.TestCoverage, error) {
	data, err := os.ReadFile(outputFile)
	if err != nil && os.IsNotExist(err) {
		return core.NewTestCoverage(), nil // Tests aren't required to produce coverage files.
	} else if err != nil {
		return core.NewTestCoverage(), err
	}
	return parseTestCoverage(target, data, run)
}

// parseTestCoverage parses coverage from loaded data.
func parseTestCoverage(target *core.BuildTarget, data []byte, run int) (*core.TestCoverage, error) {
	coverage := core.NewTestCoverage()
	if len(data) == 0 {
		return coverage, fmt.Errorf("Empty coverage output")
	} else if looksLikeGoCoverageResults(data) {
		return coverage, parseGoCoverageResults(target, coverage, data)
	} else if looksLikeGcovCoverageResults(data) {
		return coverage, parseGcovCoverageResults(target, coverage, data)
	} else if looksLikeIstanbulCoverageResults(data) {
		return coverage, parseIstanbulCoverageResults(target, coverage, data, run)
	} else {
		return coverage, parseXMLCoverageResults(target, coverage, data)
	}
}

// AddOriginalTargetsToCoverage adds empty coverage entries for any files covered by the original
// query that we haven't discovered through tests to the overall report.
// The coverage reports only contain information about files that were covered during
// tests, so it's important that we identify anything with zero coverage here.
func AddOriginalTargetsToCoverage(state *core.BuildState, includeAllFiles bool) {
	recordedCoverage := state.Coverage
	state.Coverage = core.TestCoverage{Tests: recordedCoverage.Tests, Files: map[string][]core.LineCoverage{}}
	mergeCoverage(state, recordedCoverage, collectCoverageFiles(state, includeAllFiles))
}

// collectCoverageFiles collects all the coverage files for all original targets.
// It returns a map of filename to whether it should be considered or not (test targets
// and test only targets are not considered for coverage metrics).
func collectCoverageFiles(state *core.BuildState, includeAllFiles bool) map[string]bool {
	doneTargets := map[*core.BuildTarget]bool{}
	coverageFiles := map[string]bool{}
	for _, label := range state.ExpandAllOriginalLabels() {
		collectAllFiles(state, state.Graph.TargetOrDie(label), coverageFiles, includeAllFiles, true, doneTargets)
	}
	return coverageFiles
}

// collectAllFiles collects all the source files from a single target
func collectAllFiles(state *core.BuildState, target *core.BuildTarget, coverageFiles map[string]bool, includeAllFiles, deps bool, doneTargets map[*core.BuildTarget]bool) {
	if !doneTargets[target] {
		doneTargets[target] = true
		for _, path := range target.AllSourcePaths(state.Graph) {
			if hasCoverageExtension(state, path) {
				coverageFiles[path] = !target.IsTest() && !target.TestOnly // Skip test source files from actual coverage display
			}
		}
		if deps {
			for _, dep := range target.ExternalDependencies() {
				collectAllFiles(state, dep, coverageFiles, includeAllFiles, deps, doneTargets)
			}
		}
	}
}

// hasCoverageExtension returns true if the given filename has an extension that we consider as coverable.
func hasCoverageExtension(state *core.BuildState, filename string) bool {
	extension := filepath.Ext(filename)
	for _, ext := range state.Config.Cover.FileExtension {
		if ext == extension {
			return true
		}
	}
	return false
}

// mergeCoverage merges recorded coverage with the list of all existing files.
func mergeCoverage(state *core.BuildState, recordedCoverage core.TestCoverage, coverageFiles map[string]bool) {
	doneFiles := map[string]bool{}
	for file, coverage := range recordedCoverage.Files {
		if coverageFiles[file] {
			state.Coverage.Files[file] = coverage
			doneFiles[file] = true
		}
	}
	// For any files left over now, enter them in as 100% uncovered.
	// This is pessimistic but there's not much we can do at this point.
	for file, include := range coverageFiles {
		if include && !doneFiles[file] {
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

// countLines returns the number of lines in a file.
func countLines(path string) int {
	data, _ := os.ReadFile(path)
	return bytes.Count(data, []byte{'\n'})
}

// WriteCoverageToFileOrDie writes the collected coverage data to a file in JSON format. Dies on failure.
func WriteCoverageToFileOrDie(coverage core.TestCoverage, filename string, incrementalStats *IncrementalStats) {
	out := jsonCoverage{Tests: map[string]map[string]string{}}
	allowedFiles := coverage.OrderedFiles()

	for label, coverage := range coverage.Tests {
		out.Tests[label.String()] = convertCoverage(coverage, allowedFiles)
	}

	out.Files = convertCoverage(coverage.Files, allowedFiles)
	out.Stats = getStats(coverage)
	out.Stats.Incremental = incrementalStats
	out.Stats.CoverageByDirectory = getDirectoryCoverage(coverage)
	if b, err := json.MarshalIndent(out, "", "    "); err != nil {
		log.Fatalf("Failed to encode json: %s", err)
	} else if err := os.WriteFile(filename, b, 0644); err != nil {
		log.Fatalf("Failed to write coverage results to %s: %s", filename, err)
	}
}

// WriteXMLCoverageToFileOrDie writes the collected coverage data to a file in XML format. Dies on failure.
func WriteXMLCoverageToFileOrDie(sources []core.BuildLabel, coverage core.TestCoverage, filename string) {
	data := coverageResultToXML(sources, coverage)

	if err := os.WriteFile(filename, data, 0644); err != nil {
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

type lines struct {
	covered        int
	totalCoverable int
}

func (lns lines) getPercentage() float32 {
	if lns.totalCoverable == 0 {
		return 0.0
	}
	return 100.0 * float32(lns.covered) / float32(lns.totalCoverable)
}

func getDirectoryCoverage(coverage core.TestCoverage) map[string]float32 {
	dirCoverage := make(map[string]float32)
	linesByDir := make(map[string]*lines)

	for file, coverage := range coverage.Files {
		covered, total := CountCoverage(coverage)
		dirpath := filepath.Dir(file)

		if _, exists := linesByDir[dirpath]; exists {
			linesByDir[dirpath].covered += covered
			linesByDir[dirpath].totalCoverable += total
		} else {
			linesByDir[dirpath] = &lines{covered, total}
		}
		dirCoverage[dirpath] = linesByDir[dirpath].getPercentage()
	}

	return dirCoverage
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
	TotalCoverage       float32            `json:"total_coverage"`
	CoverageByFile      map[string]float32 `json:"coverage_by_file"`
	CoverageByDirectory map[string]float32 `json:"coverage_by_directory"`
	Incremental         *IncrementalStats  `json:"incremental,omitempty"`
}

// IncrementalStats is a struct describing summarised stats for incremental coverage info.
type IncrementalStats struct {
	ModifiedFiles int     `json:"modified_files"`
	ModifiedLines int     `json:"modified_lines"`
	CoveredLines  int     `json:"covered_lines"`
	Percentage    float32 `json:"percentage"`
}

// RemoveFilesFromCoverage removes any files with extensions matching the given set from coverage.
func RemoveFilesFromCoverage(coverage core.TestCoverage, extensions []string, globs []string) {
	for _, files := range coverage.Tests {
		removeFilesFromCoverage(files, extensions)
		removeGlobsFromCoverage(files, globs)
	}
	removeFilesFromCoverage(coverage.Files, extensions)
	removeGlobsFromCoverage(coverage.Files, globs)
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

func removeGlobsFromCoverage(files map[string][]core.LineCoverage, globs []string) {
	for filename := range files {
		for _, glob := range globs {
			if ok, _ := fs.Match(glob, filename); ok {
				delete(files, filename)
			}
		}
	}
}

// CalculateIncrementalStats works out incremental coverage statistics based on a set of changed lines from files.
func CalculateIncrementalStats(state *core.BuildState, lines map[string][]int) *IncrementalStats {
	return calculateIncrementalStats(state, state.Coverage, lines, collectCoverageFiles(state, true))
}

func calculateIncrementalStats(state *core.BuildState, coverage core.TestCoverage, lines map[string][]int, files map[string]bool) *IncrementalStats {
	stats := &IncrementalStats{}
	for file, lines := range lines {
		// Include all files except those explicitly marked as test targets.
		if include, present := files[file]; include || !present {
			// Only include files that are marked as coverable types.
			if hasCoverageExtension(state, file) {
				stats.ModifiedFiles++
				if coverage, present := coverage.Files[file]; present {
					for _, line := range lines {
						if line-1 < len(coverage) { // -1 because they're 1-indexed.
							if c := coverage[line-1]; c == core.Covered {
								stats.ModifiedLines++
								stats.CoveredLines++
							} else if c == core.Uncovered || c == core.Unreachable {
								stats.ModifiedLines++
							} // Non-executable lines don't count here.
						}
					}
				} else {
					// Don't know anything about it, assume all lines are uncovered.
					stats.ModifiedLines += len(lines)
				}
			}
		}
	}
	if stats.ModifiedLines > 0 {
		stats.Percentage = 100.0 * float32(stats.CoveredLines) / float32(stats.ModifiedLines)
	}
	return stats
}
