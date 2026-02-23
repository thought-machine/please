package query

import (
	"fmt"
	"io"

	"github.com/thought-machine/please/src/core"
)

// RuntimeDeps prints all transitive run-time dependencies of a set of targets.
func RuntimeDeps(out io.Writer, state *core.BuildState, labels []core.BuildLabel) {
	done := map[core.BuildLabel]bool{}
	for _, label := range labels {
		runtimeDeps(out, state, state.Graph.TargetOrDie(label), done)
	}
}

// runtimeDeps prints all transitive run-time dependencies of a target, except for those that have
// already been printed (which are given by the keys of "done").
func runtimeDeps(out io.Writer, state *core.BuildState, target *core.BuildTarget, done map[core.BuildLabel]bool) {
	for l := range target.IterAllRuntimeDependencies(state.Graph) {
		if done[l] {
			continue
		}
		done[l] = true
		fmt.Fprintf(out, "%s\n", l.String())
	}
}
