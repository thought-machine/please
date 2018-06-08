package test

import (
	"io/ioutil"
	"path/filepath"

	"core"
	"fs"
)

func CopySurefireXmlFilesToDir(graph *core.BuildGraph, surefireDir string) {
	for _, target := range graph.AllTargets() {
		if target.IsTest {
			outputDir := target.OutDir()
			if !core.PathExists(outputDir) {
				// Unable to find tests
				continue
			}
			fs.Walk(outputDir, func(path string, isDir bool) error {
				if !isDir {
					bytes, _ := ioutil.ReadFile(path)
					if looksLikeJUnitXMLTestResults(bytes) {
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
