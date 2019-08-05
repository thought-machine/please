// Code for parsing Go's coverage output.
//
// Go comes with a built-in coverage tool and a package to parse its output format. <3
// Its format is actually rather richer than ours and can handle sub-line coverage etc.
// We may look into taking more advantage of that later...

package test

import (
	"bytes"

	"github.com/peterebden/tools/cover"

	"github.com/thought-machine/please/src/core"
)

func looksLikeGoCoverageResults(results []byte) bool {
	return bytes.HasPrefix(results, []byte("mode: "))
}

func parseGoCoverageResults(target *core.BuildTarget, coverage *core.TestCoverage, data []byte) error {
	profiles, err := cover.ParseReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	for _, profile := range profiles {
		coverage.Files[profile.FileName] = parseBlocks(profile.Blocks)
	}
	coverage.Tests[target.Label] = coverage.Files
	return nil
}

func parseBlocks(blocks []cover.ProfileBlock) []core.LineCoverage {
	if len(blocks) == 0 {
		return nil
	}
	lastLine := blocks[len(blocks)-1].EndLine
	ret := make([]core.LineCoverage, lastLine)
	for _, block := range blocks {
		for line := block.StartLine - 1; line < block.EndLine; line++ {
			if block.Count > 0 {
				ret[line] = core.Covered
			} else {
				ret[line] = core.Uncovered
			}
		}
	}
	return ret
}
