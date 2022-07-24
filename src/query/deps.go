package query

import (
	"fmt"
	"io"
	"os"

	"github.com/thought-machine/please/src/core"
)

// Deps prints all transitive dependencies of a set of targets.
func Deps(state *core.BuildState, labels []core.BuildLabel, hidden bool, targetLevel int) {
	deps(os.Stdout, state, labels, hidden, targetLevel)
}

func deps(out io.Writer, state *core.BuildState, labels []core.BuildLabel, hidden bool, targetLevel int) {
	done := map[core.BuildLabel]bool{}
	for _, label := range labels {
		printTarget(out, state, state.Graph.TargetOrDie(label), "", done, hidden, 0, targetLevel)
	}
}

func printTarget(out io.Writer, state *core.BuildState, target *core.BuildTarget, indent string, done map[core.BuildLabel]bool, hidden bool, currentLevel int, targetLevel int) {
	levelLimitReached := targetLevel != -1 && currentLevel == targetLevel
	if done[target.Label] || levelLimitReached {
		return
	}

	if state.ShouldInclude(target) && (hidden || !target.HasParent()) {
		fmt.Fprintf(out, "%s%s\n", indent, target)
		indent += "  "
		currentLevel++
	}
	done[target.Label] = true

	for _, dep := range target.Dependencies() {
		printTarget(out, state, dep, indent, done, hidden, currentLevel, targetLevel)
	}
}
