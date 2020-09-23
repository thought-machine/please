package query

import (
	"fmt"
	"strings"

	"github.com/thought-machine/please/src/core"
)

// Filter takes the list of BuildLabels and checks which ones match the label selectors passed in.
func Filter(state *core.BuildState, labels core.BuildLabels, showHidden bool) {
	// Eventually this could be more clever...
	matcher := state.ShouldInclude

	for _, label := range labels {
		if showHidden || !strings.HasPrefix(label.Name, "_") {
			if matcher(state.Graph.TargetOrDie(label)) {
				fmt.Println(label)
			}
		}
	}
}
