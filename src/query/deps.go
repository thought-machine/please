package query

import (
	"core"
	"fmt"
)

// Deps prints all transitive dependencies of a set of targets.
func Deps(state *core.BuildState, labels []core.BuildLabel, unique bool, level int) {
	targets := map[*core.BuildTarget]bool{}
	for _, label := range labels {
		printTarget(state, state.Graph.TargetOrDie(label), "", targets, unique, 0, level)
	}
}

func printTarget(state *core.BuildState, target *core.BuildTarget, indent string, targets map[*core.BuildTarget]bool,
	unique bool, level int, targetLevel int) {

	if unique && targets[target] {
		return
	}
	targets[target] = true
	if state.ShouldInclude(target) {
		fmt.Printf("%s%s\n", indent, target.Label)
	}
	if !unique {
		indent = indent + "  "
	}

	if level == targetLevel {
		return
	}
	level += 1

	for _, dep := range target.Dependencies() {
		printTarget(state, dep, indent, targets, unique, level, targetLevel)
	}
}
