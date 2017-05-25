package core

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// Used to identify the fixed part at the start of a glob pattern.
var initialFixedPart = regexp.MustCompile("([^\\*]+)/(.*)")

// IsGlob returns true if the given pattern requires globbing (i.e. contains characters that would be expanded by it)
func IsGlob(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}

// Glob implements matching using Go's built-in filepath.Glob, but extends it to support
// Ant-style patterns using **.
func Glob(rootPath string, includes, prefixedExcludes, excludes []string, includeHidden bool) []string {
	filenames := []string{}
	for _, include := range includes {
		matches, err := glob(rootPath, include, includeHidden, prefixedExcludes)
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

func glob(rootPath, pattern string, includeHidden bool, excludes []string) ([]string, error) {
	// Go's Glob function doesn't handle Ant-style ** patterns. Do it ourselves if we have to,
	// but we prefer not since our solution will have to do a potentially inefficient walk.
	if !strings.Contains(pattern, "*") {
		return []string{path.Join(rootPath, pattern)}, nil
	} else if !strings.Contains(pattern, "**") {
		return filepath.Glob(path.Join(rootPath, pattern))
	}

	// Optimisation: when we have a fixed part at the start, add that to the root path.
	// e.g. glob(["src/**/*"]) should start walking in src and not at the current directory,
	// because it can't possibly match anything else at that level.
	// Can be quite important in cases where it would descend into a massive node_modules tree
	// or similar, which leads to a big slowdown since it's synchronous with parsing
	// (ideally it would not be of course, but that's a more complex change and this is useful anyway).
	submatches := initialFixedPart.FindStringSubmatch(pattern)
	if submatches != nil {
		rootPath = path.Join(rootPath, submatches[1])
		pattern = submatches[2]
	}
	if !PathExists(rootPath) {
		return nil, nil
	}

	matches := []string{}
	// Turn the pattern into a regex. Oh dear...
	pattern = "^" + path.Join(rootPath, pattern) + "$"
	pattern = strings.Replace(pattern, "*", "[^/]*", -1)        // handle single (all) * components
	pattern = strings.Replace(pattern, "[^/]*[^/]*", ".*", -1)  // handle ** components
	pattern = strings.Replace(pattern, "/.*/", "/(?:.*/)?", -1) // allow /**/ to match nothing
	pattern = strings.Replace(pattern, "+", "\\+", -1)          // escape +
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return matches, err
	}

	err = filepath.Walk(rootPath, func(name string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if name != rootPath && IsPackage(name) {
				return filepath.SkipDir // Can't glob past a package boundary
			} else if !includeHidden && strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir // Don't descend into hidden directories
			} else if shouldExcludeMatch(name, excludes) {
				return filepath.SkipDir
			}
		} else if regex.MatchString(name) && !shouldExcludeMatch(name, excludes) {
			matches = append(matches, name)
		}
		return nil
	})
	return matches, err
}

// Memoize this to cut down on filesystem operations
var isPackageMemo = map[string]bool{}
var isPackageMutex sync.RWMutex

// IsPackage returns true if the given directory name is a package (i.e. contains a build file)
func IsPackage(name string) bool {
	isPackageMutex.RLock()
	ret, present := isPackageMemo[name]
	isPackageMutex.RUnlock()
	if present {
		return ret
	}
	ret = isPackageInternal(name)
	isPackageMutex.Lock()
	isPackageMemo[name] = ret
	isPackageMutex.Unlock()
	return ret
}

func isPackageInternal(name string) bool {
	for _, buildFileName := range State.Config.Please.BuildFileName {
		if FileExists(path.Join(name, buildFileName)) {
			return true
		}
	}
	return false
}
