package parse

import (
	"fmt"
	"strings"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
)

// Max levenshtein distance that we'll suggest at.
const maxSuggestionDistance = 3

// suggestTargets suggests the targets in the given package that might be misspellings of
// the requested one.
func suggestTargets(pkg *core.Package, label, dependent core.BuildLabel) string {
	// The initial haystack only contains target names
	haystack := []string{}
	for _, t := range pkg.AllTargets() {
		haystack = append(haystack, fmt.Sprintf("//%s:%s", pkg.Name, t.Label.Name))
	}
	msg := cli.PrettyPrintSuggestion(label.String(), haystack, maxSuggestionDistance)
	if pkg.Name != dependent.PackageName {
		return msg
	}
	// Use relative package labels where possible.
	return strings.ReplaceAll(msg, "//"+pkg.Name+":", ":")
}

// buildFileNames returns a descriptive version of the configured BUILD file names.
func buildFileNames(l []string) string {
	if len(l) == 1 {
		return l[0]
	}
	return strings.Join(l[:len(l)-1], ", ") + " or " + l[len(l)-1]
}
