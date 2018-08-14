package query

import (
	"fmt"
	"strings"

	"core"
)

// AllTargets simply prints all the targets according to some expression.
func AllTargets(graph *core.BuildGraph, labels core.BuildLabels, showHidden bool) {
	for _, label := range labels {
		if showHidden || !strings.HasPrefix(label.Name, "_") {
			fmt.Printf("%s\n", label)
		}
	}
}
