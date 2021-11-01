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
	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/plz"
	"github.com/thought-machine/please/src/utils"
)

var log = logging.MustGetLogger("format")

// Format reformats the given BUILD files to their canonical version.
// It either prints the reformatted versions to stdout or rewrites the files in-place.
// If no files are given then all BUILD files under the repo root are discovered.
// The returned bool is true if any changes were needed.
func Format(config *core.Configuration, filenames []string, rewrite, quiet bool) (bool, error) {
	if len(filenames) == 0 {
		return formatAll(plz.FindAllBuildFiles(config, core.RepoRoot, ""), rewrite, quiet)
	}
	ch := make(chan string)
	go func() {
		for _, filename := range filenames {
			ch <- filename
		}
		close(ch)
	}()
	return formatAll(ch, rewrite, quiet)
}

func formatAll(filenames <-chan string, rewrite, quiet bool) (bool, error) {
	changed := false
	for filename := range filenames {
		c, err := format(filename, rewrite, quiet)
		if err != nil {
			return changed, err
		}
		changed = changed || c
	}
	return changed, nil
}

func format(filename string, rewrite, quiet bool) (bool, error) {
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
		log.Debug("%s is already in canonical format", filename)
		return false, nil
	} else if !rewrite {
		log.Debug("%s is not in canonical format", filename)
		if !quiet {
			os.Stdout.Write(after)
		}
		return true, nil
	}
	log.Info("Rewriting %s into canonical format", filename)
	info, err := os.Stat(filename)
	if err != nil {
		return true, err
	}
	return true, fs.WriteFile(bytes.NewReader(after), filename, info.Mode())
}
