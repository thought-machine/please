package fs

import (
	"os"
	"strings"
)

// PathSeparators are the characters that separate elements of a path. Windows accepts either,
// and both turn up in practice: its own APIs return backslashes, but plenty of paths reaching
// us were written with forward slashes.
const PathSeparators = `/\`

// ExeSuffix is what an executable's filename ends in. Windows will not run a file without
// it, whatever the file actually contains.
const ExeSuffix = ".exe"

// defaultPathExt is used when PATHEXT isn't set in the environment; it matches what Windows
// itself defaults to.
const defaultPathExt = ".COM;.EXE;.BAT;.CMD"

// ExecutableNames returns the filenames to try when searching the path for an executable
// called name. Windows decides what is executable by extension, so a bare name like "bash"
// has to be tried as "bash.exe", "bash.cmd" and so on. The bare name is returned first, since
// callers may already have passed a full filename.
func ExecutableNames(name string) []string {
	pathExt := os.Getenv("PATHEXT")
	if pathExt == "" {
		pathExt = defaultPathExt
	}
	names := []string{name}
	for _, ext := range strings.Split(pathExt, ";") {
		if ext = strings.TrimSpace(ext); ext != "" {
			names = append(names, name+strings.ToLower(ext))
		}
	}
	return names
}
