// Code for parsing XML coverage output (eg. Java or Python).

package test

import "encoding/xml"
import "strings"

import (
	"core"
	"fmt"
	"strconv"
	"time"
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
func CoverageResultToXML(sources []core.BuildLabel, coverage core.TestCoverage) []byte {
	linesValid := 0
	linesCovered := 0
	validFiles := coverage.OrderedFiles()

	// get the string representative of sources
	var sourcesAsStr []string
	for _, source := range sources {
		sourcesAsStr = append(sourcesAsStr, source.PackageName + ":" + source.Name)
	}

	// Get the list of packages for <package> tag in the coverage xml file
	var packages []Package
	for label, coverage := range coverage.Tests {
		packageName := strings.Replace(label.String(), ":", "/", -1)

		// Get the list of classes for <class> tag in the coverage xml file
		var classes []Class
		classLineRateTotal := float32(0)
		for className, lineCover := range coverage {
			// Do not include files in coverage report if its not valid
			if !shouldInclude(className, validFiles) {
				continue
			}

			lines, covered, total := GetLineCoverageInfo(lineCover)
			classLineRate := float32(covered) / float32(total)

			cls := Class{Name:className, Filename: className,
						 Lines:lines, LineRate:formatFloatPrecision(classLineRate, 4)}

		    classes = append(classes, cls)
		    classLineRateTotal += classLineRate
			linesValid += total
			linesCovered += covered
		}

		pkgLineRate := float32(classLineRateTotal) / float32(len(classes))

		if len(classes) != 0 {
			pkg := Package{Name:packageName, Classes:classes, LineRate:formatFloatPrecision(pkgLineRate, 4)}
			packages = append(packages, pkg)
		}
	}

	topLevelLineRate := float32(linesCovered) / float32(linesValid)

	// Create the coverage object based on the data collected
	coverageObj := Coverage{Packages:packages, LineRate:formatFloatPrecision(topLevelLineRate, 4),
						    LinesCovered:int64(linesCovered), LinesValid: int64(linesValid),
						    Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
							Sources:sourcesAsStr}

    // Parse struct to xml bytes
	xmlBytes, err := xml.MarshalIndent(coverageObj, "", "	")
    if err != nil {
		log.Fatalf("Failed to parse to xml: %s", err)
	}
	covReport := []byte(xml.Header + string(xmlBytes))
	return covReport
}

func shouldInclude(filename string, ValidFile []string) bool {
	for _, f := range ValidFile {
		if filename == f {
			return true
		}
	}

	return false
}

// Get the line coverage info, returns: list of lines covered, num of covered lines, and total valid lines
func GetLineCoverageInfo(lineCover []core.LineCoverage) ([]Line, int, int) {
	var lines []Line
	covered := 0
	total := 0

	for index, status := range lineCover {
		if status == 3 {
			line := Line{Hits:1, Number:index}
			lines = append(lines, line)
			covered += 1
			total += 1
		} else if status == 2 {
			line := Line{Hits:0, Number:index}
			lines = append(lines, line)
			total += 1
		}
	}

	return lines, covered, total
}

// format the float32 numbers to a specific precision
func formatFloatPrecision(val float32, precision int) float32 {
	precString := fmt.Sprintf("%v", precision)
	strVal := fmt.Sprintf("%." + precString + "f", val)

	i, err := strconv.ParseFloat(strVal, 32)

	if err != nil {
		log.Fatalf("Failed to parse float: %s", err)
	}

	return float32(i)
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


// Coverage struct for writing to xml file
type Coverage struct {
	XMLName         xml.Name  `xml:"coverage"`
	LineRate        float32   `xml:"line-rate,attr"`
	BranchRate      float32   `xml:"branch-rate,attr"`
	LinesCovered    int64     `xml:"lines-covered,attr"`
	LinesValid      int64     `xml:"lines-valid,attr"`
	BranchesCovered int64     `xml:"branches-covered,attr"`
	BranchesValid   int64     `xml:"branches-valid,attr"`
	Complexity      float32   `xml:"complexity,attr"`
	Version         string    `xml:"version,attr"`
	Timestamp       int64     `xml:"timestamp,attr"`
	Sources         []string   `xml:"sources>source"`
	Packages        []Package `xml:"packages>package"`
}

type Package struct {
	Name       string  `xml:"name,attr"`
	LineRate   float32 `xml:"line-rate,attr"`
	BranchRate float32 `xml:"branch-rate,attr"`
	Complexity float32 `xml:"complexity,attr"`
	Classes    []Class `xml:"classes>class"`
	LineCount  int64   `xml:"line-count,attr"`
	LineHits   int64   `xml:"line-hits,attr"`
}

type Class struct {
	Name       string   `xml:"name,attr"`
	Filename   string   `xml:"filename,attr"`
	LineRate   float32  `xml:"line-rate,attr"`
	BranchRate float32  `xml:"branch-rate,attr"`
	Complexity float32  `xml:"complexity,attr"`
	Methods    []Method `xml:"methods>method"`
	Lines      []Line   `xml:"lines>line"`
}

type Method struct {
	Name       string  `xml:"name,attr"`
	Signature  string  `xml:"signature,attr"`
	LineRate   float32 `xml:"line-rate,attr"`
	BranchRate float32 `xml:"branch-rate,attr"`
	Complexity float32 `xml:"complexity,attr"`
	Lines      []Line  `xml:"lines>line"`
	LineCount  int64   `xml:"line-count,attr"`
	LineHits   int64   `xml:"line-hits,attr"`
}

type Line struct {
	Number int   `xml:"number,attr"`
	Hits   int64 `xml:"hits,attr"`
}
