package query

import (
	"fmt"
	"github.com/thought-machine/please/src/core"
)

// Deps prints all transitive dependencies of a set of targets.
func Deps(state *core.BuildState, labels []core.BuildLabel, unique, hidden bool, targetLevel int) {
	done := map[core.BuildLabel]bool{}
	for _, label := range labels {
		printTarget(state, state.Graph.TargetOrDie(label), "", done, unique, hidden, 0, targetLevel)
	}
}

func printTarget(state *core.BuildState, target *core.BuildTarget, indent string, done map[core.BuildLabel]bool,
	unique, hidden bool, currentLevel int, targetLevel int) {
	if unique && done[target.Label] {
		return
	}

	done[target.Label] = true
	if state.ShouldInclude(target) {
		if parent := target.Parent(state.Graph); hidden || parent == target || parent == nil {
			fmt.Printf("%s%s\n", indent, target.Label)
		} else if !done[parent.Label] {
			fmt.Printf("%s%s\n", indent, parent)
			done[parent.Label] = true
		}
	}
	if !unique {
		indent = indent + "  "
	}

	// access the level of dependency, as default is -1 which prints out everything
	if targetLevel != -1 && currentLevel == targetLevel {
		return
	}

	currentLevel++

	for _, dep := range target.Dependencies() {
		printTarget(state, dep, indent, done, unique, hidden, currentLevel, targetLevel)
	}
}
