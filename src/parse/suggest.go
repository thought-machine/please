package parse

import (
	"fmt"
	"strings"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/utils"
)

// Max levenshtein distance that we'll suggest at.
const maxSuggestionDistance = 3

// suggestTargets suggests the targets in the given package that might be misspellings of
// the requested one.
func suggestTargets(pkg *core.Package, label, dependor core.BuildLabel) string {
	// The initial haystack only contains target names
	haystack := []string{}
	for _, t := range pkg.AllTargets() {
		haystack = append(haystack, fmt.Sprintf("//%s:%s", pkg.Name, t.Label.Name))
	}
	msg := utils.PrettyPrintSuggestion(label.String(), haystack, maxSuggestionDistance)
	if pkg.Name != dependor.PackageName {
		return msg
	}
	// Use relative package labels where possible.
	return strings.Replace(msg, "//"+pkg.Name+":", ":", -1)
}
