// Package bazel provides some Bazel compatibility shims.
package bazel

import "embed"

//go:embed *.build_defs
var files embed.FS

// AllFiles returns all the embedded files as a map of name to contents.
func AllFiles() map[string][]byte {
	m := map[string][]byte{}
	entries, err := files.ReadDir(".")
	if err != nil {
		panic(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		data, err := files.ReadFile(name)
		if err != nil {
			panic(err)
		}
		m[name] = data
	}
	return m
}
