// Package utils contains various utility functions and whatnot.
package utils

import (
	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
)

var log = logging.MustGetLogger("utils")

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
