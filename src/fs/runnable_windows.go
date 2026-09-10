package fs

import (
	"os"
	"strings"
)

// ExplainUnrunnable returns extra context for a file that could not be executed, or an empty
// string if there is nothing useful to add.
//
// Windows decides what is runnable by extension, and Go's exec package enforces that: a file
// whose name has no extension in PATHEXT will not run even when handed its full path, and the
// error says it was "not found in %PATH%" - which is baffling when the file is plainly there.
// Windows itself is happy to execute it; only the lookup refuses.
//
// The usual cause is a build rule that named its output after the rule, as most language
// plugins do, without adding the suffix Windows needs.
func ExplainUnrunnable(path string) string {
	if path == "" || !PathExists(path) {
		return ""
	}
	for _, name := range ExecutableNames("") {
		if name != "" && strings.HasSuffix(strings.ToLower(path), name) {
			return ""
		}
	}
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		return ""
	}
	return "\n" + path + " exists, but its name has no extension Windows will run; something has to produce it as " + path + ExeSuffix + " instead"
}
