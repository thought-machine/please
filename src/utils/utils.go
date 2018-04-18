// Package utils contains various utility functions and whatnot.
package utils

import (
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/op/go-logging.v1"

	"cli"
	"core"
	"fs"
)

var log = logging.MustGetLogger("utils")

// FindAllSubpackages finds all packages under a particular path.
// Used to implement rules with ... where we need to know all possible packages
// under that location.
func FindAllSubpackages(config *core.Configuration, rootPath string, prefix string) <-chan string {
	ch := make(chan string)
	go func() {
		if rootPath == "" {
			rootPath = "."
		}
		if err := fs.Walk(rootPath, func(name string, isDir bool) error {
			basename := path.Base(name)
			if name == core.OutDir || (isDir && strings.HasPrefix(basename, ".") && name != ".") {
				return filepath.SkipDir // Don't walk output or hidden directories
			} else if isDir && !strings.HasPrefix(name, prefix) && !strings.HasPrefix(prefix, name) {
				return filepath.SkipDir // Skip any directory without the prefix we're after (but not any directory beneath that)
			} else if isABuildFile(basename, config) && !isDir {
				dir, _ := path.Split(name)
				ch <- strings.TrimRight(dir, "/")
			} else if cli.ContainsString(name, config.Parse.ExperimentalDir) {
				return filepath.SkipDir // Skip the experimental directory if it's set
			}
			// Check against blacklist
			for _, dir := range config.Parse.BlacklistDirs {
				if dir == basename || strings.HasPrefix(name, dir) {
					return filepath.SkipDir
				}
			}
			return nil
		}); err != nil {
			log.Fatalf("Failed to walk tree under %s; %s\n", rootPath, err)
		}
		close(ch)
	}()
	return ch
}

// isABuildFile returns true if given filename is a build file name.
func isABuildFile(name string, config *core.Configuration) bool {
	for _, buildFileName := range config.Parse.BuildFileName {
		if name == buildFileName {
			return true
		}
	}
	return false
}
