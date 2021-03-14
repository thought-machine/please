package glob

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Globber memoizes state to help speed up globbing.
type Globber struct {
	buildFileNames []string

	dirs            map[string][]os.DirEntry
	packageSearches map[string]bool
}

type matcher struct {
	includes []*pattern
	excludes []*pattern
}

type pattern struct {
	parts     []*globPart
	isExclude bool
}

type globPart struct {
	literal      string
	regex        *regexp.Regexp
	isDoubleStar bool
}

func (part *globPart) match(name string) bool {
	if part.regex != nil {
		return part.regex.MatchString(name)
	}
	return part.literal == name
}

// New createa a new globber
func New(buildFileNames []string) *Globber {
	return &Globber{
		buildFileNames:  buildFileNames,
		dirs:            map[string][]os.DirEntry{},
		packageSearches: map[string]bool{},
	}
}

// Attempts to match the pattern returning any patterns that would match sub-directories
func (p *pattern) match(name string) (bool, *pattern) {
	part := p.parts[0]
	lastPart := len(p.parts) == 1

	next := &pattern{parts: p.parts[1:]}
	if lastPart {
		next = nil
	}

	// If the part is a literal match for that name, easy
	if part.literal == name {
		return lastPart, next
	}
	// ** can match multiple dirs so we return this pattern again plus a pattern matching the next part
	if part.isDoubleStar {
		nextMatches := false
		if next != nil {
			nextMatches, _ = next.match(name)
		}
		return lastPart || nextMatches, p
	}

	// Otherwise it might be a wildcard match
	if part.match(name) {
		return lastPart, next
	}

	// Single part excludes match all the way down
	if p.isExclude && len(p.parts) == 1 {
		return false, p
	}
	return false, nil
}

func (g *Globber) match(m *matcher, name string) (bool, *matcher) {
	nextMatcher := new(matcher)

	incMatch := false
	for _, inc := range m.includes {
		match, patterns := inc.match(name)
		if patterns != nil {
			nextMatcher.includes = append(nextMatcher.includes, patterns)
		}
		if match {
			incMatch = true
		}
	}

	for _, excl := range m.excludes {
		match, patterns := excl.match(name)
		if patterns != nil {
			nextMatcher.excludes = append(nextMatcher.excludes, patterns)
		}
		if match {
			return false, nil
		}
	}

	return incMatch, nextMatcher
}

func toRegexString(pattern string) string {
	pattern = "^" + pattern + "$"
	pattern = strings.ReplaceAll(pattern, "+", "\\+")   // escape +
	pattern = strings.ReplaceAll(pattern, ".", "\\.")   // escape .
	pattern = strings.ReplaceAll(pattern, "?", ".")     // match ? as any single char
	pattern = strings.ReplaceAll(pattern, "*", "[^/]*") // handle single (all) * components
	return pattern
}

func IsGlob(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}

func toPatterns(globs []string, isExclude bool) []*pattern {
	patterns := make([]*pattern, 0, len(globs))
	for _, glob := range globs {
		patterns = append(patterns, toPattern(glob, isExclude))
	}
	return patterns
}

func toPattern(glob string, isExclude bool) *pattern {
	partStrings := strings.Split(glob, string(filepath.Separator))
	return expandDoubleStars(&pattern{isExclude: isExclude}, partStrings)
}

func expandDoubleStars(soFar *pattern, rest []string) *pattern {
	if len(rest) == 0 {
		return soFar
	}

	next := rest[0]
	// **.txt -> **/*.txt && *.txt
	if next != "**" && strings.HasPrefix(next, "**") {
		tail := strings.TrimPrefix(next, "**")
		soFar.parts = append(soFar.parts, compileGlobPart("**"), compileGlobPart(fmt.Sprintf("*%s", tail)))
	} else {
		soFar.parts = append(soFar.parts, compileGlobPart(next))
	}
	return expandDoubleStars(soFar, rest[1:])
}

func compileGlobPart(part string) *globPart {
	if part == "**" {
		return &globPart{isDoubleStar: true}
	}
	if IsGlob(part) {
		return &globPart{
			regex: regexp.MustCompile(toRegexString(part)),
		}
	}
	return &globPart{literal: part}
}

func (g *Globber) isBuildFile(name string) bool {
	for _, buildFileName := range g.buildFileNames {
		if name == buildFileName {
			return true
		}
	}
	return false
}

// Glob globs a directory based on the include and exclude patterns
func (g *Globber) Glob(root string, includeHidden bool, includes, excludes []string) ([]string, error) {
	return g.glob(root, ".", includeHidden, &matcher{
		includes: toPatterns(includes, false),
		excludes: toPatterns(excludes, true),
	})
}

func (g *Globber) glob(root, path string, includeHidden bool, m *matcher) ([]string, error) {
	dirEntries, err := g.readDir(filepath.Join(root, path))
	if err != nil {
		return nil, err
	}
	return g.globEntries(root, path, includeHidden, m, dirEntries)
}

func (g *Globber) readDir(path string) ([]os.DirEntry, error) {
	if entries, ok := g.dirs[path]; ok {
		return entries, nil
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	g.dirs[path] = entries
	return entries, nil
}

func (g *Globber) isPackage(match string) (bool, error) {
	if isPkg, ok := g.packageSearches[match]; ok {
		return isPkg, nil
	}
	entries, err := g.readDir(match)
	if err != nil {
		return false, err
	}

	// Check for a build file in this dir before decending into the tree
	for _, entry := range entries {
		if !entry.IsDir() && g.isBuildFile(entry.Name()) {
			g.packageSearches[match] = true
			return true, nil
		}
	}
	for _, entry := range entries {
		if entry.IsDir() {
			isPkg, err := g.isPackage(filepath.Join(match, entry.Name()))
			if err != nil {
				return false, err
			}
			if isPkg {
				g.packageSearches[match] = true
				return true, nil
			}
		}
	}
	g.packageSearches[match] = false
	return false, nil
}

func (g *Globber) globEntries(root, path string, includeHidden bool, matcher *matcher, entries []os.DirEntry) ([]string, error) {
	var matches []string
	for _, entry := range entries {
		// If we encounter a BUILD file, don't match this dir
		if g.isBuildFile(entry.Name()) {
			if path == "." {
				continue
			}
			return []string{}, nil
		}

		if !includeHidden && strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		matchPath := filepath.Join(path, entry.Name())
		entryPath := filepath.Join(root, matchPath)
		doesMatch, newMatcher := g.match(matcher, entry.Name())
		if doesMatch {
			if entry.IsDir() {
				if isPkg, err := g.isPackage(entryPath); err != nil {
					return nil, err
				} else if isPkg {
					continue
				}
			}
			matches = append(matches, matchPath)
		}

		// If the pattern could match more, and the entry is a directory, search for the extra matches
		if newMatcher != nil && entry.IsDir() {
			entryMatches, err := g.glob(root, matchPath, includeHidden, newMatcher)
			if err != nil {
				return nil, err
			}
			matches = append(matches, entryMatches...)
		}
	}
	return matches, nil
}
