// Package format does autoformatting of BUILD files.
//
// It is based on a mildly modified version of buildifier; that supports fstrings
// but not some of the other dialetical differences (e.g. type annotations).
package format

import (
	"bytes"
	"os"
	"sync/atomic"

	"github.com/bazelbuild/buildtools/build"
	"golang.org/x/sync/errgroup"

	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/plz"
)

var log = logging.Log

// Format reformats the given BUILD files to their canonical version.
// It either prints the reformatted versions to stdout or rewrites the files in-place.
// If no files are given then all BUILD files under the repo root are discovered.
// The returned bool is true if any changes were needed.
func Format(config *core.Configuration, filenames []string, rewrite, quiet bool) (bool, error) {
	if len(filenames) == 0 {
		return formatAll(plz.FindAllBuildFiles(config, core.RepoRoot, ""), config.Please.NumThreads, rewrite, quiet)
	}
	ch := make(chan string)
	go func() {
		for _, filename := range filenames {
			ch <- filename
		}
		close(ch)
	}()
	return formatAll(ch, config.Please.NumThreads, rewrite, quiet)
}

func formatAll(filenames <-chan string, parallelism int, rewrite, quiet bool) (bool, error) {
	var changed int64
	var g errgroup.Group
	limiter := make(chan struct{}, parallelism)
	for filename := range filenames {
		filename := filename
		g.Go(func() error {
			limiter <- struct{}{}
			defer func() { <-limiter }()
			c, err := format(filename, rewrite, quiet)
			if c {
				atomic.AddInt64(&changed, 1)
			}
			return err
		})
	}
	err := g.Wait()
	return changed > 0, err
}

func format(filename string, rewrite, quiet bool) (bool, error) {
	before, err := os.ReadFile(filename)
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
