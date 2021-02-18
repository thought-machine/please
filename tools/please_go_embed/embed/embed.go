// Package embed implements parsing of embed directives in Go files.
package embed

import (
	"fmt"
	"go/build"
	"path"
	"path/filepath"
	"strings"
)

// Cfg is the structure of a Go embedcfg file.
type Cfg struct {
	Patterns map[string][]string
	Files    map[string]string
}

// Parse parses the given files and returns the embed information in them.
func Parse(gofiles []string) (*Cfg, error) {
	cfg := &Cfg{
		Patterns: map[string][]string{},
		Files:    map[string]string{},
	}
	for _, dir := range dirs(gofiles) {
		pkg, err := build.ImportDir(dir, build.ImportComment)
		if err != nil {
			return nil, err
		}
		// We munge all patterns together at this point, if a file is in our input sources we want to know about it regardless.
		for _, pattern := range append(append(pkg.EmbedPatterns, pkg.TestEmbedPatterns...), pkg.XTestEmbedPatterns...) {
			paths, err := relglob(dir, pattern)
			if err != nil {
				return nil, err
			}
			cfg.Patterns[pattern] = paths
			for _, p := range paths {
				cfg.Files[p] = path.Join(dir, p)
			}
		}
	}
	return cfg, nil
}

func dirs(files []string) []string {
	dirs := []string{}
	seen := map[string]bool{}
	for _, f := range files {
		if dir := path.Dir(f); !seen[dir] {
			dirs = append(dirs, dir)
			seen[dir] = true
		}
	}
	return dirs
}

func relglob(dir, pattern string) ([]string, error) {
	paths, err := filepath.Glob(path.Join(dir, pattern))
	if err == nil && len(paths) == 0 {
		return nil, fmt.Errorf("pattern %s: no matching paths found", pattern)
	}
	for i, p := range paths {
		paths[i] = strings.TrimLeft(strings.TrimPrefix(p, dir), string(filepath.Separator))
	}
	return paths, err
}
