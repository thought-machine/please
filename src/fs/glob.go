package fs

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar"
	"github.com/karrick/godirwalk"
)

// IsGlob returns true if the given pattern requires globbing (i.e. contains characters that would be expanded by it)
func IsGlob(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}

// Glob implements matching using Go's built-in filepath.Glob, but extends it to support
// Ant-style patterns using **.
func Glob(buildFileNames []string, rootPath string, includes, prefixedExcludes, excludes []string, includeHidden bool) []string {
	subPackages, err := findSubPackages(rootPath, buildFileNames)
	if err != nil {
		panic(err)
	}

	var filenames []string
	for _, include := range includes {
		matches, err := glob(rootPath, include, prefixedExcludes, subPackages)
		if err != nil {
			panic(err)
		}
		for _, filename := range matches {
			if !includeHidden {
				// Exclude hidden & temporary files
				_, file := path.Split(filename)
				if strings.HasPrefix(file, ".") || (strings.HasPrefix(file, "#") && strings.HasSuffix(file, "#")) {
					continue
				}
			}
			if strings.HasPrefix(filename, rootPath) && rootPath != "" {
				filename = filename[len(rootPath)+1:] // +1 to strip the slash too
			}
			if !shouldExcludeMatch(filename, excludes) {
				filenames = append(filenames, filename)
			}
		}
	}
	return filenames
}

func shouldExcludeMatch(match string, excludes []string) bool {
	for _, excl := range excludes {
		if strings.ContainsRune(match, '/') && !strings.ContainsRune(excl, '/') {
			match = path.Base(match)
		}
		if matches, err := filepath.Match(excl, match); matches || err != nil {
			return true
		}
	}
	return false
}

func glob(rootPath, pattern string, excludes []string, subPackages []string) ([]string, error) {
	// Go's Glob function doesn't handle Ant-style ** patterns. Do it ourselves if we have to,
	// but we prefer not since our solution will have to do a potentially inefficient walk.
	if !strings.Contains(pattern, "*") {
		return []string{path.Join(rootPath, pattern)}, nil
	} else if !strings.Contains(pattern, "**") {
		return filepath.Glob(path.Join(rootPath, pattern))
	}

	globMatches, err := doublestar.Glob(path.Join(rootPath, pattern))
	if err != nil {
		return nil, err
	}

	var matches []string
	for _, m := range globMatches {
		if isInPackages(m, subPackages) {
			continue
		}
		if shouldExcludeMatch(m, excludes) {
			continue
		}
		matches = append(matches, m)
	}
	return matches, nil
}

// Walk implements an equivalent to filepath.Walk.
// It's implemented over github.com/karrick/godirwalk but the provided interface doesn't use that
// to make it a little easier to handle.
func Walk(rootPath string, callback func(name string, isDir bool) error) error {
	return WalkMode(rootPath, func(name string, isDir bool, mode os.FileMode) error {
		return callback(name, isDir)
	})
}

// WalkMode is like Walk but the callback receives an additional type specifying the file mode type.
// N.B. This only includes the bits of the mode that determine the mode type, not the permissions.
func WalkMode(rootPath string, callback func(name string, isDir bool, mode os.FileMode) error) error {
	// Compatibility with filepath.Walk which allows passing a file as the root argument.
	if info, err := os.Lstat(rootPath); err != nil {
		return err
	} else if !info.IsDir() {
		return callback(rootPath, false, info.Mode())
	}
	return godirwalk.Walk(rootPath, &godirwalk.Options{Callback: func(name string, info *godirwalk.Dirent) error {
		return callback(name, info.IsDir(), info.ModeType())
	}})
}

func findSubPackages(rootPath string, buildFileNames []string,) ([]string, error) {
	ms, err := doublestar.Glob(path.Join(rootPath, "*/**"))
	if err != nil {
		return nil, err
	}

	var subPackages []string
	for _, m := range ms {
		if isBuildFile(buildFileNames, m) {
			subPackages = append(subPackages, filepath.Dir(m))
		}
	}
	return subPackages, nil
}

// IsPackage returns true if the given directory name is a package (i.e. contains a build file)
func isBuildFile(buildFileNames []string, name string) bool {
	for _, buildFileName := range buildFileNames {
		if strings.HasSuffix(name, buildFileName) {
			return true
		}
	}
	return false
}

func isInPackages(name string, packages []string) bool {
	for _, p := range packages {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}
