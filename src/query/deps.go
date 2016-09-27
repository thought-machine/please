package query

import "core"
import "fmt"

// QueryDeps prints all transitive dependencies of a set of targets.
func QueryDeps(state *core.BuildState, labels []core.BuildLabel) {
	for _, label := range labels {
		printTarget(state, state.Graph.TargetOrDie(label), "")
	}
}

func printTarget(state *core.BuildState, target *core.BuildTarget, indent string) {
	if target.ShouldInclude(state.Include, state.Exclude) {
		fmt.Printf("%s%s\n", indent, target.Label)
	}
	for _, dep := range target.Dependencies() {
		printTarget(state, dep, indent+"  ")
	}
}
