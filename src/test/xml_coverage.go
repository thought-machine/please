// Code for parsing XML coverage output (eg. Java or Python).

package test

import "encoding/xml"
import "strings"

import "core"

func parseXMLCoverageResults(target *core.BuildTarget, coverage *core.TestCoverage, data []byte) error {
	xcoverage := xmlCoverage{}
	if err := xml.Unmarshal(data, &xcoverage); err != nil {
		return err
	}
	for _, pkg := range xcoverage.Packages.Package {
		for _, cls := range pkg.Classes.Class {
			if strings.HasPrefix(cls.Filename, core.RepoRoot) {
				cls.Filename = cls.Filename[len(core.RepoRoot):]
			}
			// There can be multiple classes per file so we must merge here, not overwrite.
			coverage.Files[cls.Filename] = core.MergeCoverageLines(coverage.Files[cls.Filename], parseXMLLines(cls.Lines.Line))
		}
	}
	coverage.Tests[target.Label] = coverage.Files
	return nil
}

func parseXMLLines(lines []xmlCoverageLine) []core.LineCoverage {
	ret := []core.LineCoverage{}
	for _, line := range lines {
		for i := len(ret) + 1; i < line.Number; i++ {
			ret = append(ret, core.NotExecutable)
		}
		if line.Hits > 0 {
			ret = append(ret, core.Covered)
		} else {
			ret = append(ret, core.Uncovered)
		}
	}
	return ret
}

// Note that this is based off coverage.py's format, which is originally a Java format
// so some of the structures are a little awkward (eg. 'classes' actually refer to Python modules, not classes).
type xmlCoverage struct {
	Packages struct {
		Package []struct {
			Classes struct {
				Class []struct {
					LineRate float32 `xml:"line-rate,attr"`
					Filename string  `xml:"filename,attr"`
					Name     string  `xml:"name,attr"`
					Lines    struct {
						Line []xmlCoverageLine `xml:"line"`
					} `xml:"lines"`
				} `xml:"class"`
			} `xml:"classes"`
		} `xml:"package"`
	} `xml:"packages"`
}

type xmlCoverageLine struct {
	Hits   int `xml:"hits,attr"`
	Number int `xml:"number,attr"`
}
