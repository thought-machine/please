package query

import (
	"fmt"

	"github.com/thought-machine/please/src/core"
)

// Filter takes the list of BuildLabels and checks which ones match the label selectors passed in.
func Filter(state *core.BuildState, labels core.BuildLabels) {

	// Eventually this could be more clever...
	matcher := state.ShouldInclude

	for _, label := range labels {
		if matcher(state.Graph.TargetOrDie(label)) {
			fmt.Println(label)
		}
	}
}
