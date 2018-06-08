package test

import (
	"io/ioutil"
	"path/filepath"

	"core"
	"fs"
)

func CopySurefireXmlFilesToDir(graph *core.BuildGraph, surefireDir string) {
	log.Infof("Copy files to %s", surefireDir)
	for _, target := range graph.AllTargets() {
		if target.IsTest {
			log.Infof("Checking %s", target.OutDir())
			outputDir := target.OutDir()
			if !core.PathExists(outputDir) {
				// Unable to find tests
        		continue
			}
			fs.Walk(outputDir, func(path string, isDir bool) error {
				if !isDir {
					bytes, _ := ioutil.ReadFile(path)
					if looksLikeJUnitXMLTestResults(bytes) {
			    		log.Infof("Found target with tests: %v", target)
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
