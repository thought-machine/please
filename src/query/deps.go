package query

import "core"
import "fmt"

// Deps prints all transitive dependencies of a set of targets.
func Deps(state *core.BuildState, labels []core.BuildLabel, unique bool) {
	targets := map[*core.BuildTarget]bool{}
	for _, label := range labels {
		printTarget(state, state.Graph.TargetOrDie(label), "", targets, unique)
	}
}

func printTarget(state *core.BuildState, target *core.BuildTarget, indent string, targets map[*core.BuildTarget]bool, unique bool) {
	if unique && targets[target] {
		return
	}
	targets[target] = true
	if target.ShouldInclude(state.Include, state.Exclude) {
		fmt.Printf("%s%s\n", indent, target.Label)
	}
	if !unique {
		indent = indent + "  "
	}
	for _, dep := range target.Dependencies() {
		printTarget(state, dep, indent, targets, unique)
	}
}
