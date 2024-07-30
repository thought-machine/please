package query

import (
	"fmt"
	"io"
	"os"

	"github.com/thought-machine/please/src/core"
)

// Deps prints all transitive dependencies of a set of targets.
func Deps(state *core.BuildState, labels []core.BuildLabel, hidden bool, targetLevel int, formatdot bool) {
	out := os.Stdout
	if formatdot {
		fmt.Fprintf(out, "digraph deps {\n")
		fmt.Fprintf(out, "  fontname=\"Helvetica,Arial,sans-serif\"\n")
		fmt.Fprintf(out, "  node [fontname=\"Helvetica,Arial,sans-serif\"]\n")
		fmt.Fprintf(out, "  edge [fontname=\"Helvetica,Arial,sans-serif\"]\n")
		fmt.Fprintf(out, "  rankdir=\"LR\"\n")
	}
	done := map[core.BuildLabel]bool{}
	for _, label := range labels {
		if formatdot {
			fmt.Fprintf(out, "  subgraph \"%s\" {\n", label)
			printTargetDot(out, state, state.Graph.TargetOrDie(label), nil, done, hidden, 0, targetLevel)
			fmt.Fprintf(out, "  }\n")
		} else {
			printTarget(out, state, state.Graph.TargetOrDie(label), "", done, hidden, 0, targetLevel)
		}
	}
	if formatdot {
		fmt.Fprintf(out, "}\n")
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
	if target.Subrepo != nil && target.Subrepo.Target != nil {
		printTarget(out, state, target.Subrepo.Target, indent, done, hidden, currentLevel, targetLevel)
	}
}

func printTargetDot(out io.Writer, state *core.BuildState, target *core.BuildTarget, parent *core.BuildTarget, done map[core.BuildLabel]bool, hidden bool, currentLevel int, targetLevel int) {
	levelLimitReached := targetLevel != -1 && currentLevel == targetLevel

	if state.ShouldInclude(target) && (hidden || !target.HasParent()) {
		if !done[target.Label] {
			shape := "ellipse"
			if target.IsFilegroup {
				shape = "folder"
			} else if target.IsRemoteFile {
				shape = "octagon"
			} else if target.IsTextFile {
				shape = "note"
			} else if target.IsBinary {
				shape = "component"
			}
			fmt.Fprintf(out, "   node [shape=%s] \"%s\";\n", shape, target)
		}
		if parent != nil {
			fmt.Fprintf(out, "   \"%s\" -> \"%s\";\n", parent, target)
		}
		currentLevel++
	}
	if done[target.Label] || levelLimitReached {
		return
	}
	done[target.Label] = true

	for _, dep := range target.Dependencies() {
		printTargetDot(out, state, dep, target, done, hidden, currentLevel, targetLevel)
	}
}
