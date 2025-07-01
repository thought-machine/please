// Code for parsing XML coverage output (eg. Java or Python).

package test

import (
	"encoding/xml"
	"math"
	"path/filepath"
	"strings"
	"time"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
)

func parseXMLCoverageResults(target *core.BuildTarget, coverage *core.TestCoverage, data []byte) error {
	xcoverage := xmlCoverage{}
	if err := xml.Unmarshal(data, &xcoverage); err != nil {
		return err
	}
	for _, pkg := range xcoverage.Packages.Package {
		for _, cls := range pkg.Classes.Class {
			filename := strings.TrimPrefix(cls.Filename, core.RepoRoot)
			// There can be multiple classes per file so we must merge here, not overwrite.
			coverage.Files[filename] = core.MergeCoverageLines(coverage.Files[filename], parseXMLLines(cls.Lines.Line))
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

// Covert the Coverage result to XML bytes, ready to be write to file
func coverageResultToXML(sources []core.BuildLabel, coverage core.TestCoverage) []byte {
	linesValid := 0
	linesCovered := 0
	validFiles := coverage.OrderedFiles()

	// get the string representative of sources
	sourcesAsStr := make([]string, len(sources))
	for i, source := range sources {
		sourcesAsStr[i] = filepath.Join(core.RepoRoot, source.PackageName)
	}

	// Get the list of packages for <package> tag in the coverage xml file
	var packages []pkg
	for label, coverage := range coverage.Tests {
		packageName := label.String()

		// Get the list of classes for <class> tag in the coverage xml file
		var classes []class
		classLineRateTotal := 0.0
		for className, lineCover := range coverage {
			// Do not include files in coverage report if its not valid
			if !cli.ContainsString(className, validFiles) {
				continue
			}

			lines, covered, total := getLineCoverageInfo(lineCover)
			classLineRate := float64(covered) / float64(total)

			cls := class{Name: className, Filename: className,
				Lines: lines, LineRate: formatFloatPrecision(classLineRate, 4)}

			classes = append(classes, cls)
			classLineRateTotal += classLineRate
			linesValid += total
			linesCovered += covered
		}

		pkgLineRate := classLineRateTotal / float64(len(classes))

		if len(classes) != 0 {
			pkg := pkg{Name: packageName, Classes: classes, LineRate: formatFloatPrecision(pkgLineRate, 4)}
			packages = append(packages, pkg)
		}
	}

	topLevelLineRate := float64(linesCovered) / float64(linesValid)

	// Create the coverage object based on the data collected
	coverageObj := coverageType{Packages: packages, LineRate: formatFloatPrecision(topLevelLineRate, 4),
		LinesCovered: linesCovered,
		LinesValid:   linesValid,
		Timestamp:    int(time.Now().UnixNano()) / int(time.Millisecond),
		Sources:      sourcesAsStr}

	// Serialise struct to xml bytes
	xmlBytes, err := xml.MarshalIndent(coverageObj, "", "	")
	if err != nil {
		log.Fatalf("Failed to parse to xml: %s", err)
	}
	covReport := []byte(xml.Header + string(xmlBytes))
	return covReport
}

// Get the line coverage info, returns: list of lines covered, num of covered lines, and total valid lines
func getLineCoverageInfo(lineCover []core.LineCoverage) ([]line, int, int) {
	var lines []line
	covered := 0
	total := 0

	for index, status := range lineCover {
		switch status {
		case core.Covered:
			line := line{Hits: 1, Number: index}
			lines = append(lines, line)
			covered++
			total++
		case core.Uncovered:
			line := line{Hits: 0, Number: index}
			lines = append(lines, line)
			total++
		}
	}

	return lines, covered, total
}

// format the float64 numbers to a specific precision
func formatFloatPrecision(val float64, precision int) float64 {
	unit := math.Pow10(precision)
	return math.Round(val*unit) / unit
}

// Note that this is based off coverage.py's format, which is originally a Java format
// so some of the structures are a little awkward (eg. 'classes' actually refer to Python modules, not classes).
type xmlCoverage struct {
	Packages struct {
		Package []struct {
			Classes struct {
				Class []struct {
					LineRate float64 `xml:"line-rate,attr"`
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

// Coverage struct for writing to xml file
type coverageType struct {
	XMLName         xml.Name `xml:"coverage"`
	LineRate        float64  `xml:"line-rate,attr"`
	BranchRate      float64  `xml:"branch-rate,attr"`
	LinesCovered    int      `xml:"lines-covered,attr"`
	LinesValid      int      `xml:"lines-valid,attr"`
	BranchesCovered int      `xml:"branches-covered,attr"`
	BranchesValid   int      `xml:"branches-valid,attr"`
	Complexity      float64  `xml:"complexity,attr"`
	Version         string   `xml:"version,attr"`
	Timestamp       int      `xml:"timestamp,attr"`
	Sources         []string `xml:"sources>source"`
	Packages        []pkg    `xml:"packages>package"`
}

type pkg struct {
	Name       string  `xml:"name,attr"`
	LineRate   float64 `xml:"line-rate,attr"`
	BranchRate float64 `xml:"branch-rate,attr"`
	Complexity float64 `xml:"complexity,attr"`
	Classes    []class `xml:"classes>class"`
	LineCount  int     `xml:"line-count,attr"`
	LineHits   int     `xml:"line-hits,attr"`
}

type class struct {
	Name       string   `xml:"name,attr"`
	Filename   string   `xml:"filename,attr"`
	LineRate   float64  `xml:"line-rate,attr"`
	BranchRate float64  `xml:"branch-rate,attr"`
	Complexity float64  `xml:"complexity,attr"`
	Methods    []method `xml:"methods>method"`
	Lines      []line   `xml:"lines>line"`
}

type method struct {
	Name       string  `xml:"name,attr"`
	Signature  string  `xml:"signature,attr"`
	LineRate   float64 `xml:"line-rate,attr"`
	BranchRate float64 `xml:"branch-rate,attr"`
	Complexity float64 `xml:"complexity,attr"`
	Lines      []line  `xml:"lines>line"`
	LineCount  int     `xml:"line-count,attr"`
	LineHits   int     `xml:"line-hits,attr"`
}

type line struct {
	Number int `xml:"number,attr"`
	Hits   int `xml:"hits,attr"`
}
