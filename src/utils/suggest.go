package utils

import (
	"sort"

	"github.com/texttheater/golang-levenshtein/levenshtein"
)

// Suggest implements levenshtein-based suggestions on a sequence of items.
func Suggest(needle string, haystack []string, maxSuggestionDistance int) []string {
	r := []rune(needle)
	options := make(suggestions, 0, len(haystack))
	for _, straw := range haystack {
		distance := levenshtein.DistanceForStrings(r, []rune(straw), levenshtein.DefaultOptions)
		if len(straw) > 0 && distance <= maxSuggestionDistance {
			options = append(options, suggestion{s: straw, dist: distance})
		}
	}
	sort.Sort(options)
	ret := make([]string, len(options))
	for i, o := range options {
		ret[i] = o.s
	}
	return ret
}

// PrettyPrintSuggestion implements levenshtein-based suggestions on a sequence of items and
// produces a single message from them.
func PrettyPrintSuggestion(needle string, haystack []string, maxSuggestionDistance int) string {
	options := Suggest(needle, haystack, maxSuggestionDistance)
	if len(options) == 0 {
		return ""
	}
	// Obviously there's now more code to pretty-print the suggestions than to do the calculation...
	msg := "\nMaybe you meant "
	for i, o := range options {
		if i > 0 {
			if i < len(options)-1 {
				msg += " , " // Leave a space before the comma so you can select them without getting the question mark
			} else {
				msg += " or "
			}
		}
		msg += o
	}
	return msg + " ?" // Leave a space so you can select them without getting the question mark
}

type suggestion struct {
	s    string
	dist int
}
type suggestions []suggestion

func (s suggestions) Len() int           { return len(s) }
func (s suggestions) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s suggestions) Less(i, j int) bool { return s[i].dist < s[j].dist }
