// Package format does autoformatting of BUILD files.
//
// It is based on a mildly modified version of buildifier; that supports fstrings
// but not some of the other dialetical differences (e.g. type annotations).
package format

import (
	"bytes"
	"io/ioutil"
	"os"

	"github.com/bazelbuild/buildtools/build"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/utils"
)

// Format reformats the given BUILD files to their canonical version.
// It either prints the reformatted versions to stdout or rewrites the files in-place.
// If no files are given then all BUILD files under the repo root are discovered.
// The returned bool is true if any changes were needed.
func Format(config *core.Configuration, filenames []string, rewrite bool) (bool, error) {
	if len(filenames) == 0 {
		return formatAll(utils.FindAllSubpackages(config, core.RepoRoot, ""), rewrite)
	}
	ch := make(chan string)
	go func() {
		for _, filename := range filenames {
			ch <- filename
		}
		close(ch)
	}()
	return formatAll(ch, rewrite)
}

func formatAll(filenames <-chan string, rewrite bool) (bool, error) {
	changed := false
	for filename := range filenames {
		c, err := format(filename, rewrite)
		if err != nil {
			return changed, err
		}
		changed = changed || c
	}
	return changed, nil
}

func format(filename string, rewrite bool) (bool, error) {
	before, err := ioutil.ReadFile(filename)
	if err != nil {
		return true, err
	}
	f, err := build.ParseBuild(filename, before)
	if err != nil {
		return true, err
	}
	after := build.Format(f)
	if bytes.Equal(before, after) {
		return false, nil
	} else if !rewrite {
		os.Stdout.Write(after)
		return true, nil
	}
	info, err := os.Stat(filename)
	if err != nil {
		return true, err
	}
	return true, fs.WriteFile(bytes.NewReader(after), filename, info.Mode())
}
