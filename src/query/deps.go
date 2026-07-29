package query

import (
	"fmt"
	"io"
	"strings"

	"github.com/thought-machine/please/src/core"
)

// Deps prints all transitive dependencies of a set of targets.
func Deps(out io.Writer, state *core.BuildState, labels []core.BuildLabel, hidden bool, targetLevel int, formatdot bool) {
	if formatdot {
		fmt.Fprintf(out, "digraph deps {\n")
		fmt.Fprintf(out, "  fontname=\"Helvetica,Arial,sans-serif\"\n")
		fmt.Fprintf(out, "  node [fontname=\"Helvetica,Arial,sans-serif\"]\n")
		fmt.Fprintf(out, "  edge [fontname=\"Helvetica,Arial,sans-serif\"]\n")
		fmt.Fprintf(out, "  rankdir=\"LR\"\n")
	}
	visit := func(dep, parent *core.BuildTarget, level int) {
		if formatdot {
			printTargetDot(out, dep, parent)
		} else {
			printTarget(out, dep, level)
		}
	}
	done := map[core.BuildLabel]bool{}
	for _, label := range labels {
		walkDeps(state, state.Graph.TargetOrDie(label), done, targetLevel, 0, hidden, visit)
	}
	if formatdot {
		fmt.Fprintf(out, "}\n")
	}
}

// DepsLabels returns all transitive dependencies of a set of targets in traversal order.
func DepsLabels(state *core.BuildState, labels []core.BuildLabel, hidden bool, targetLevel int) core.BuildLabels {
	ret := core.BuildLabels{}
	visit := func(dep, parent *core.BuildTarget, level int) {
		ret = append(ret, dep.Label)
	}
	done := map[core.BuildLabel]bool{}
	for _, label := range labels {
		walkDeps(state, state.Graph.TargetOrDie(label), done, targetLevel, 0, hidden, visit)
	}
	return ret
}

// walkDeps looks at all the deps of the given target & recurses into them,
// calling visit for each dependency as it's discovered.
func walkDeps(
	state *core.BuildState,
	target *core.BuildTarget,
	done map[core.BuildLabel]bool,
	targetLevel, currentLevel int,
	hidden bool,
	visit func(dep, parent *core.BuildTarget, level int),
) {
	if currentLevel == targetLevel {
		return
	}
	for _, l := range target.DeclaredDependencies() {
		dep := state.Graph.TargetOrDie(l)
		for _, l := range dep.ProvideFor(target) {
			if !state.ShouldInclude(dep) || done[l] {
				continue // target is filtered out
			}
			done[l] = true
			if dep := state.Graph.TargetOrDie(l); hidden || !dep.HasParent() {
				// dep is to be visited; either we're including hidden deps or it has no parent (i.e. is not hidden)
				visit(dep, target, currentLevel)
				walkDeps(state, dep, done, targetLevel, currentLevel+1, hidden, visit)
			} else if dep.Label.Parent() == target.Label.Parent() {
				// This is a hidden dependency of the current target, recurse without increasing depth
				walkDeps(state, dep, done, targetLevel, currentLevel, hidden, visit)
			} else {
				walkDeps(state, dep, done, targetLevel, currentLevel+1, hidden, visit)
			}
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
