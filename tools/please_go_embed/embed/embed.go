// Package embed implements parsing of embed directives in Go files.
package embed

import (
	"path"
	"path/filepath"
	"go/build"
)

// EmbedCfg is the structure of a Go embedcfg file.
type EmbedCfg struct {
	Patterns map[string][]string
	Files    map[string]string
}

// Parse parses the given files and returns the embed information in them.
func Parse(gofiles []string) (*EmbedCfg, error) {
	cfg := &EmbedCfg{
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
			paths, err := filepath.Glob(pattern)
			if err != nil {
				return nil, err
			}
			cfg.Patterns[pattern] = paths
			for _, path := range paths {
				cfg.Files[path] = path  // Go seems to use an absolute path here... hope it's not necessary?
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
