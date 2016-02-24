package parse

import (
	"fmt"
	"sort"

	"github.com/texttheater/golang-levenshtein/levenshtein"

	"core"
)

// Max levenshtein distance that we'll suggest at.
const maxSuggestionDistance = 3

// suggestTargets suggests the targets in the given package that might be misspellings of
// the requested one.
func suggestTargets(pkg *core.Package, label, dependor core.BuildLabel) string {
	r := []rune(label.Name)
	options := make(suggestions, 0, len(pkg.Targets))
	for t := range pkg.Targets {
		distance := levenshtein.DistanceForStrings(r, []rune(t), levenshtein.DefaultOptions)
		if distance <= maxSuggestionDistance {
			options = append(options, suggestion{name: t, dist: distance})
		}
	}
	if len(options) == 0 {
		return ""
	}
	sort.Sort(options)
	// Obviously there's now more code to pretty-print the suggestions than to do the calculation...
	msg := "\nMaybe you meant "
	for i, o := range options {
		if i > 0 {
			if i < len(options)-1 {
				msg += ", "
			} else {
				msg += " or "
			}
		}
		if pkg.Name == dependor.PackageName {
			msg += ":" + o.name
		} else {
			msg += fmt.Sprintf("//%s:%s", pkg.Name, o.name)
		}
	}
	return msg + " ?" // Leave a space so you can select them without getting the question mark
}

type suggestion struct {
	name string
	dist int
}
type suggestions []suggestion

func (s suggestions) Len() int           { return len(s) }
func (s suggestions) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s suggestions) Less(i, j int) bool { return s[i].dist < s[j].dist }
