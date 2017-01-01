package parse

import (
	"fmt"
	"strings"

	"core"
	"utils"
)

// Max levenshtein distance that we'll suggest at.
const maxSuggestionDistance = 3

// suggestTargets suggests the targets in the given package that might be misspellings of
// the requested one.
func suggestTargets(pkg *core.Package, label, dependor core.BuildLabel) string {
	// The initial haystack only contains target names
	haystack := make([]string, 0, len(pkg.Targets))
	for t := range pkg.Targets {
		haystack = append(haystack, fmt.Sprintf("//%s:%s", pkg.Name, t))
	}
	msg := utils.SuggestMessage(label.String(), haystack, maxSuggestionDistance)
	if pkg.Name != dependor.PackageName {
		return msg
	}
	// Use relative package labels where possible.
	return strings.Replace(msg, "//"+pkg.Name+":", ":", -1)
}
