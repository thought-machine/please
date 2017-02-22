// Code for parsing Istanbul JSON coverage results.

package test

import (
	"bytes"
	"encoding/json"
	"strings"

	"core"
)

func looksLikeIstanbulCoverageResults(results []byte) bool {
	// This works because this is the only JSON format that we accept. If we accept another
	// we may need to get cleverer about it.
	return bytes.HasPrefix(results, []byte("{"))
}

func parseIstanbulCoverageResults(target *core.BuildTarget, coverage *core.TestCoverage, data []byte) error {
	files := map[string]istanbulFile{}
	if err := json.Unmarshal(data, &files); err != nil {
		return err
	}
	for filename, file := range files {
		coverage.Files[sanitiseFileName(target, filename)] = file.toLineCoverage()
	}
	coverage.Tests[target.Label] = coverage.Files
	return nil
}

type istanbulFile struct {
	// StatementMap identifies the start and end for each statement.
	StatementMap map[string]istanbulLocation `json:"statementMap"`
	// Statements identifies the covered statements.
	Statements map[string]int `json:"s"`
}

// An istanbulLocation defines a start/end location in the instrumented source code.
type istanbulLocation struct {
	Start istanbulLineLocation `json:"start"`
	End   istanbulLineLocation `json:"end"`
}

// An istanbulLineLocation defines a single location in the instrumented source code.
type istanbulLineLocation struct {
	Column int `json:"column"`
	Line   int `json:"line"`
}

// toLineCoverage coverts this object to our internal format.
func (file *istanbulFile) toLineCoverage() []core.LineCoverage {
	ret := make([]core.LineCoverage, file.maxLineNumber())
	for statement, count := range file.Statements {
		val := core.Uncovered
		if count > 0 {
			val = core.Covered
		}
		s := file.StatementMap[statement]
		for i := s.Start.Line; i <= s.End.Line; i++ {
			ret[i-1] = val // -1 because 1-indexed
		}
	}
	return ret
}

// maxLineNumber returns the highest line number present in this file.
func (file *istanbulFile) maxLineNumber() int {
	max := 0
	for _, s := range file.StatementMap {
		if s.End.Line > max {
			max = s.End.Line
		}
	}
	return max
}

// sanitiseFileName strips out any build/test paths found in the given file.
func sanitiseFileName(target *core.BuildTarget, filename string) string {
	if index := strings.Index(filename, target.OutDir()); index != -1 {
		return filename[index+len(target.OutDir())+1:]
	}
	if index := strings.Index(filename, target.TmpDir()); index != -1 {
		return filename[index+len(target.TmpDir())+1:]
	}
	if index := strings.Index(filename, target.TestDir()); index != -1 {
		return filename[index+len(target.TestDir())+1:]
	}
	return filename
}
