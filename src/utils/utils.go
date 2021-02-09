// Package utils contains various utility functions and whatnot.
package utils

import (
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

var log = logging.MustGetLogger("utils")

// FindAllSubpackages finds all packages under a particular path.
// Used to implement rules with ... where we need to know all possible packages
// under that location.
func FindAllSubpackages(config *core.Configuration, rootPath, prefix string) <-chan string {
	ch := make(chan string)
	go func() {
		for filename := range FindAllBuildFiles(config, rootPath, prefix) {
			dir, _ := path.Split(filename)
			ch <- strings.TrimRight(dir, "/")
		}
		close(ch)
	}()
	return ch
}

// FindAllBuildFiles finds all BUILD files under a particular path.
// It's like FindAllSubpackages but gives the filename as well as the directory.
func FindAllBuildFiles(config *core.Configuration, rootPath, prefix string) <-chan string {
	ch := make(chan string)
	go func() {
		if rootPath == "" {
			rootPath = "."
		}
		if err := fs.Walk(rootPath, func(name string, isDir bool) error {
			basename := path.Base(name)
			if basename == core.OutDir || (isDir && strings.HasPrefix(basename, ".") && name != ".") {
				return filepath.SkipDir // Don't walk output or hidden directories
			} else if isDir && !strings.HasPrefix(name, prefix) && !strings.HasPrefix(prefix, name) {
				return filepath.SkipDir // Skip any directory without the prefix we're after (but not any directory beneath that)
			} else if config.IsABuildFile(basename) && !isDir {
				ch <- name
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

// Max returns the larger of two ints.
func Max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

// ReadingStdin returns true if any of the given build labels are reading from stdin.
func ReadingStdin(labels []core.BuildLabel) bool {
	for _, l := range labels {
		if l == core.BuildLabelStdin {
			return true
		}
	}
	return false
}

// ReadingStdin returns true if any of the given build labels are reading from stdin.
func ReadingStdinAnnnotated(labels []core.AnnotatedOutputLabel) bool {
	for _, l := range labels {
		if l.BuildLabel == core.BuildLabelStdin {
			return true
		}
	}
	return false
}

func AnnotateLabels(labels []core.BuildLabel) []core.AnnotatedOutputLabel {
	ret := make([]core.AnnotatedOutputLabel, len(labels))
	for i, l := range labels {
		ret[i] = core.AnnotatedOutputLabel{BuildLabel: l}
	}
	return ret
}

// ReadStdinLabels reads any of the given labels from stdin, if any of them indicate it
// (i.e. if ReadingStdin(labels) is true, otherwise it just returns them.
func ReadStdinLabels(labels []core.BuildLabel) []core.BuildLabel {
	if !ReadingStdin(labels) {
		return labels
	}
	ret := []core.BuildLabel{}
	for _, l := range labels {
		if l == core.BuildLabelStdin {
			for s := range cli.ReadStdin() {
				ret = append(ret, core.ParseBuildLabels([]string{s})...)
			}
		} else {
			ret = append(ret, l)
		}
	}
	return ret
}
