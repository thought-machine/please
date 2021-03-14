package fs

import fsglob "github.com/thought-machine/please/src/fs/glob"

// Glob implements matching using Go's built-in filepath.Glob, but extends it to support
// Ant-style patterns using **.
func Glob(buildFileNames []string, rootPath string, includes, excludes []string, includeHidden bool) []string {
	matches, err := fsglob.New(buildFileNames).Glob(rootPath, includeHidden, includes, excludes)
	if err != nil {
		panic(err)
	}
	return matches
}
