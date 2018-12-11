package test

import (
	"io/ioutil"
	"path/filepath"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

// CopySurefireXmlFilesToDir copies all the XML test results files into the given directory.
func CopySurefireXmlFilesToDir(state *core.BuildState, surefireDir string) {
	for _, label := range state.ExpandOriginalLabels() {
		target := state.Graph.TargetOrDie(label)
		if state.ShouldInclude(target) && target.IsTest && !target.NoTestOutput {
			if path := target.TestResultsFile(); fs.PathExists(path) {
				fs.Walk(path, func(path string, isDir bool) error {
					if !isDir {
						if bytes, _ := ioutil.ReadFile(path); looksLikeJUnitXMLTestResults(bytes) {
							surefireResult := filepath.Join(surefireDir, filepath.Base(path))
							if err := fs.CopyOrLinkFile(path, surefireResult, 0644, true, true); err != nil {
								log.Errorf("Error linking %s to %s - %s", surefireResult, path, err)
							}
						}
					}
					return nil
				})
			}
		}
	}
}
