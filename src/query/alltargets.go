package query

import (
		"strings"

	"core"
	"fmt"
)

// AllTargets simply prints all the targets according to some expression.
func AllTargets(state *core.BuildState, labels core.BuildLabels, showHidden bool) {
	for _, label := range labels {
		if showHidden || !strings.HasPrefix(label.Name, "_") {
			target := state.Graph.TargetOrDie(label)
			if state.ShouldInclude(target) {
				fmt.Printf("%s\n", target.Label)
			}
		}
	}
}
