package query

import (
	"fmt"

	"github.com/thought-machine/please/src/core"
)

// Deps prints all transitive dependencies of a set of targets.
func Deps(state *core.BuildState, labels []core.BuildLabel, hidden bool, targetLevel int) {
	done := map[core.BuildLabel]bool{}
	for _, label := range labels {
		printTarget(state, state.Graph.TargetOrDie(label), "", done, hidden, 0, targetLevel)
	}
}

func printTarget(state *core.BuildState, target *core.BuildTarget, indent string, done map[core.BuildLabel]bool, hidden bool, currentLevel int, targetLevel int) {
	if !state.ShouldInclude(target) {
		return
	}

	levelLimitReached := targetLevel != -1 && currentLevel == targetLevel
	if done[target.Label] || levelLimitReached {
		return
	}

	if hidden || !target.HasParent() {
		fmt.Printf("%s%s\n", indent, target)
		done[target.Label] = true

		indent += "  "
		currentLevel++
	}

	for _, dep := range target.Dependencies() {
		printTarget(state, dep, indent, done, hidden, currentLevel, targetLevel)
	}
}
