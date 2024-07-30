package query

import (
	"fmt"
	"io"
	"os"
	"strings"

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
	done := map[*core.BuildTarget]bool{}
	for _, label := range labels {
		deps(out, state, state.Graph.TargetOrDie(label), done, targetLevel, 0, hidden, formatdot)
	}
	if formatdot {
		fmt.Fprintf(out, "}\n")
	}
}

// deps looks at all the deps of the given target & recurses into them, printing as appropriate.
func deps(out io.Writer, state *core.BuildState, target *core.BuildTarget, done map[*core.BuildTarget]bool, targetLevel, currentLevel int, hidden, formatdot bool) {
	if done[target] || currentLevel == targetLevel {
		return
	}
	done[target] = true
	for _, dep := range target.Dependencies() {
		if !state.ShouldInclude(dep) || done[dep] {
			continue // target is filtered out
		} else if hidden || !dep.HasParent() {
			// dep is to be printed
			if formatdot {
				printTargetDot(out, dep, target)
			} else {
				printTarget(out, dep, currentLevel)
			}
			deps(out, state, dep, done, targetLevel, currentLevel+1, hidden, formatdot)
		} else {
			// If we didn't print it, we still need to recurse; we don't increase the depth though.
			deps(out, state, dep, done, targetLevel, currentLevel, hidden, formatdot)
		}
	}
}

func printTarget(out io.Writer, target *core.BuildTarget, currentLevel int) {
	indent := strings.Repeat("  ", currentLevel)
	fmt.Fprintf(out, "%s%s\n", indent, target.Label)
}

func printTargetDot(out io.Writer, target, parent *core.BuildTarget) {
	fmt.Fprintf(out, "  subgraph \"%s\" {\n", target)
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
	fmt.Fprintf(out, "   \"%s\" -> \"%s\";\n", parent, target)
	fmt.Fprintf(out, "  }\n")
}
