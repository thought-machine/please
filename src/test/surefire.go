package test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

// CopySurefireXMLFilesToDir copies all the XML test results files into the given directory.
func CopySurefireXMLFilesToDir(state *core.BuildState, surefireDir string) {
	for _, label := range state.ExpandOriginalLabels() {
		target := state.Graph.TargetOrDie(label)
		if state.ShouldInclude(target) && target.IsTest && !target.NoTestOutput {
			for i := 1; fs.PathExists(target.TestResultsFile(i)); i++ {
				copySurefireXMLtoDir(target.TestResultsFile(i), surefireDir)
			}
		}
	}
}

func copySurefireXMLtoDir(path string, surefireDir string) {
	fs.WalkMode(path, func(path string, isDir bool, mode os.FileMode) error {
		if !isDir {
			if surefireResult := filepath.Join(surefireDir, filepath.Base(path)); !fs.PathExists(surefireResult) {
				if bytes, _ := ioutil.ReadFile(path); looksLikeJUnitXMLTestResults(bytes) {
					if err := fs.CopyOrLinkFile(path, surefireResult, mode, 0644, true, true); err != nil {
						log.Errorf("Error linking %s to %s - %s", surefireResult, path, err)
					}
				}
			}
		}
		return nil
	})
}
