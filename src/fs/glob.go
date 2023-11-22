package fs

import (
	"fmt"
	iofs "io/fs"
	"path/filepath"
	"regexp"
	"strings"
)

type matcher interface {
	Match(name string) (bool, error)
}

type builtInGlob string

func (p builtInGlob) Match(name string) (bool, error) {
	matched, err := filepath.Match(string(p), name)
	if err != nil {
		return false, fmt.Errorf("failed to glob, invalid patern: %v, %w", string(p), err)
	}
	return matched, nil
}

type regexGlob struct {
	regex *regexp.Regexp
}

func (r regexGlob) Match(name string) (bool, error) {
	return r.regex.MatchString(name), nil
}

// This converts the string pattern into a matcher. A matcher can either be one of our homebrew compiled regexs that
// support ** or a matcher that uses the built in filesystem.Match functionality.
func patternToMatcher(root, pattern string) (matcher, error) {
	fullPattern := filepath.Join(root, pattern)

	// Use the built in filesystem.Match globs when not using double star as it's far more efficient
	if !strings.Contains(pattern, "**") {
		return builtInGlob(fullPattern), nil
	}
	regex, err := regexp.Compile(toRegexString(fullPattern))
	if err != nil {
		return nil, fmt.Errorf("failed to compile glob pattern %s, %w", pattern, err)
	}
	return regexGlob{regex: regex}, nil
}

func toRegexString(pattern string) string {
	pattern = "^" + pattern + "$"
	pattern = strings.ReplaceAll(pattern, "+", "\\+")         // escape +
	pattern = strings.ReplaceAll(pattern, ".", "\\.")         // escape .
	pattern = strings.ReplaceAll(pattern, "?", ".")           // match ? as any single char
	pattern = strings.ReplaceAll(pattern, "*", "[^/]*")       // handle single (all) * components
	pattern = strings.ReplaceAll(pattern, "[^/]*[^/]*", ".*") // handle ** components
	pattern = strings.ReplaceAll(pattern, "/.*/", "/(.*/)?")  // Allow ** to match zero directories
	return pattern
}

// IsGlob returns true if the given pattern requires globbing (i.e. contains characters that would be expanded by it)
func IsGlob(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}

// Glob implements matching using Go's built-in filepath.Glob, but extends it to support
// Ant-style patterns using **.
func Glob(fs iofs.FS, buildFileNames []string, rootPath string, includes, excludes []string, includeHidden bool) []string {
	return NewGlobber(fs, buildFileNames).Glob(rootPath, includes, excludes, includeHidden, true)
}

// A Globber is used to implement Glob. You can persist one for use to save repeated filesystem calls, but
// it isn't safe for use in concurrent goroutines.
type Globber struct {
	buildFileNames []string
	fs             iofs.FS
	walkedDirs     map[string]walkedDir
}

type walkedDir struct {
	fileNames, symlinks, subPackages []string
}

func Match(glob, path string) (bool, error) {
	matcher, err := patternToMatcher(".", glob)
	if err != nil {
		return false, err
	}
	return matcher.Match(path)
}

// NewGlobber creates a new Globber. You should call this rather than creating one directly (or use Glob() if you don't care).
func NewGlobber(fs iofs.FS, buildFileNames []string) *Globber {
	return &Globber{
		buildFileNames: buildFileNames,
		fs:             fs,
		walkedDirs:     map[string]walkedDir{},
	}
}

func (globber *Globber) Glob(rootPath string, includes, excludes []string, includeHidden, includeSymlinks bool) []string {
	if rootPath == "" {
		rootPath = "."
	}

	var filenames []string
	for _, include := range includes {
		mustBeValidGlobString(include)

		matches, err := globber.glob(rootPath, include, excludes, includeHidden, includeSymlinks)
		if err != nil {
			panic(fmt.Errorf("error globbing files with %v: %v", include, err))
		}
		// Remove the root path from the returned files and add them to the output
		for _, filename := range matches {
			filenames = append(filenames, strings.TrimPrefix(filename, rootPath+"/"))
		}
	}
	return filenames
}

func (globber *Globber) glob(rootPath string, glob string, excludes []string, includeHidden, includeSymlinks bool) ([]string, error) {
	p, err := patternToMatcher(rootPath, glob)
	if err != nil {
		return nil, err
	}
	walkedDir, err := globber.walkDir(rootPath)
	if err != nil {
		return nil, err
	}
	var globMatches []string

	fileNames := walkedDir.fileNames
	if includeSymlinks {
		fileNames = append(fileNames, walkedDir.symlinks...)
	}
	for _, name := range fileNames {
		if match, err := p.Match(name); err != nil {
			return nil, err
		} else if match {
			globMatches = append(globMatches, name)
		}
	}

	matches := make([]string, 0, len(globMatches))
	for _, m := range globMatches {
		if isInDirectories(m, walkedDir.subPackages) {
			continue
		}
		if !includeHidden && isHidden(m) {
			continue
		}

		shouldExclude, err := shouldExcludeMatch(rootPath, m, excludes)
		if err != nil {
			return nil, err
		}
		if shouldExclude {
			continue
		}

		matches = append(matches, m)
	}
	return matches, nil
}

func (globber *Globber) walkDir(rootPath string) (walkedDir, error) {
	if dir, present := globber.walkedDirs[rootPath]; present {
		return dir, nil
	}
	dir := walkedDir{}
	err := iofs.WalkDir(globber.fs, rootPath, func(path string, d iofs.DirEntry, err error) error {
		typeMode := mode(d.Type())
		if isBuildFile(globber.buildFileNames, path) {
			packageName := filepath.Dir(path)
			if packageName != rootPath {
				dir.subPackages = append(dir.subPackages, packageName)
				return filepath.SkipDir
			}
		}
		// Exclude plz-out
		if d.Name() == "plz-out" && rootPath == "." {
			return filepath.SkipDir
		}
		if typeMode.IsSymlink() {
			dir.symlinks = append(dir.symlinks, path)
		} else {
			dir.fileNames = append(dir.fileNames, path)
		}
		return nil
	})
	if err != nil {
		return dir, err
	}
	globber.walkedDirs[rootPath] = dir
	return dir, nil
}

func mustBeValidGlobString(glob string) {
	if glob == "" {
		panic("cannot use an empty string as a glob")
	}
}

func isBathPathOf(path string, base string) bool {
	if !strings.HasPrefix(path, base) {
		return false
	}

	rest := strings.TrimPrefix(path, base)
	return rest == "" || rest[0] == filepath.Separator
}

// shouldExcludeMatch checks if the match also matches any of the exclude patterns. If the exclude pattern is a relative
// pattern i.e. doesn't contain any /'s, then the pattern is checked against the file name part only. Otherwise the
// pattern is checked against the whole path. This is so `glob(["**/*.go"], exclude = ["*_test.go"])` will match as
// you'd expect.
func shouldExcludeMatch(root, match string, excludes []string) (bool, error) {
	for _, excl := range excludes {
		mustBeValidGlobString(excl)

		rootPath := root
		m := match

		if isBathPathOf(match, filepath.Join(root, excl)) {
			return true, nil
		}

		// If the exclude pattern doesn't contain any slashes and the match does, we only match against the base of the
		// match path.
		if strings.ContainsRune(match, '/') && !strings.ContainsRune(excl, '/') {
			m = filepath.Base(match)
			rootPath = ""
		}

		matcher, err := patternToMatcher(rootPath, excl)
		if err != nil {
			return false, err
		}

		match, err := matcher.Match(m)
		if err != nil {
			return false, err
		}
		if match {
			return true, nil
		}
	}
	return false, nil
}

// isBuildFile checks if the filename is considered a build filename
func isBuildFile(buildFileNames []string, name string) bool {
	fileName := filepath.Base(name)
	for _, buildFileName := range buildFileNames {
		if fileName == buildFileName {
			return true
		}
	}
	return false
}

// isInDirectories checks to see if the file is in any of the provided directories
func isInDirectories(name string, directories []string) bool {
	for _, dir := range directories {
		if strings.HasPrefix(name, dir+"/") || name == dir {
			return true
		}
	}
	return false
}

// isHidden checks if the file is a hidden file i.e. starts with . or, starts and ends with #.
func isHidden(name string) bool {
	file := filepath.Base(name)
	return strings.HasPrefix(file, ".") || (strings.HasPrefix(file, "#") && strings.HasSuffix(file, "#"))
}
