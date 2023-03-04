// Code for parsing gcov coverage output.

package test

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/thought-machine/please/src/core"
)

// gcovPlaceholder is used to mark empty coverage files for C++ tests when coverage is disabled for them.
var gcovPlaceholder = []byte{'g', 'c', 'o', 'v'}

func parseGcovCoverageResults(target *core.BuildTarget, coverage *core.TestCoverage, data []byte) error {
	if bytes.Equal(data, gcovPlaceholder) {
		return nil // Coverage is disabled, not an error.
	}
	// The data we have is a sequence of .gcov files smashed together.
	lines := bytes.Split(data, []byte{'\n'})
	if len(lines) == 0 {
		return fmt.Errorf("Empty coverage file")
	}
	currentFilename := ""
	for lineno, line := range lines {
		fields := bytes.Split(line, []byte{':'})
		if len(fields) < 3 {
			continue
		}
		if bytes.Equal(fields[2], []byte("Source")) {
			if len(fields) < 4 {
				return fmt.Errorf("Bad source on line %d: %s", lineno, string(line))
			}
			currentFilename = string(fields[3])
			continue
		}
		covLine, err := strconv.Atoi(strings.TrimSpace(string(fields[1])))
		if err != nil {
			return fmt.Errorf("Bad line number on line %d: %s", lineno, string(line))
		} else if covLine > 0 {
			coverage.Files[currentFilename] = append(coverage.Files[currentFilename], translateGcovCount(bytes.TrimSpace(fields[0])))
		}
	}
	return nil
}

// translateGcovCount coverts gcov's format to ours.
// AFAICT the format is:
//
//	    -: Not executable
//	#####: Not covered
//	   32: line was hit 32 times
func translateGcovCount(gcov []byte) core.LineCoverage {
	if len(gcov) > 0 && gcov[0] == '-' {
		return core.NotExecutable
	} else if i, err := strconv.Atoi(string(gcov)); err == nil && i > 0 {
		return core.Covered
	}
	return core.Uncovered
}

// looksLikeGcovCoverageResults returns true if the given data appears to be gcov results.
func looksLikeGcovCoverageResults(data []byte) bool {
	return bytes.HasPrefix(data, []byte("        -:    0:Source:")) || bytes.Equal(data, gcovPlaceholder)
}
