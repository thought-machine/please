package test

import (
	"io/ioutil"
	"path/filepath"

	"core"
	"fs"
)

// CopySurefireXmlFilesToDir copies all the XML test results files into the given directory.
func CopySurefireXmlFilesToDir(state *core.BuildState, surefireDir string) {
	for _, label := range state.ExpandOriginalLabels() {
		target := state.Graph.TargetOrDie(label)
		if state.ShouldInclude(target) && target.IsTest && !target.NoTestOutput {
			if path := target.TestResultsFile(); fs.PathExists(path) {
				bytes, _ := ioutil.ReadFile(path)
				if looksLikeJUnitXMLTestResults(bytes) {
					surefireResult := filepath.Join(surefireDir, filepath.Base(path))
					if err := fs.CopyOrLinkFile(path, surefireResult, 0644, true, true); err != nil {
						log.Errorf("Error linking %s to %s - %s", surefireResult, path, err)
					}
				}
			}
		}
	}
}
