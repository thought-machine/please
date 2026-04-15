package cli

import (
	"sort"
	"strings"

	"github.com/texttheater/golang-levenshtein/levenshtein"
)

// Suggest implements levenshtein-based suggestions on a sequence of items.
func Suggest(needle string, haystack []string, maxSuggestionDistance int) []string {
	r := []rune(needle)
	options := make([]suggestion, 0, len(haystack))
	for _, straw := range haystack {
		distance := levenshtein.DistanceForStrings(r, []rune(straw), levenshtein.DefaultOptions)
		if len(straw) > 0 && distance <= maxSuggestionDistance {
			options = append(options, suggestion{s: straw, dist: distance})
		}
	}
	sort.Slice(options, func(i, j int) bool { return options[i].dist < options[j].dist })
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
	var msgBuilder strings.Builder
	msgBuilder.WriteString("\nMaybe you meant ")
	for i, o := range options {
		if i > 0 {
			if i < len(options)-1 {
				msgBuilder.WriteString(" , ") // Leave a space before the comma so you can select them without getting the question mark
			} else {
				msgBuilder.WriteString(" or ")
			}
		}
		msgBuilder.WriteString(o)
	}
	msgBuilder.WriteString(" ?") // Leave a space so you can select them without getting the question mark
	return msgBuilder.String()
}

type suggestion struct {
	s    string
	dist int
}
