package query

import (
	"fmt"
	"path"

	"github.com/thought-machine/please/src/core"
)

// TargetOutputs prints all output files for a set of targets.
func TargetOutputs(graph *core.BuildGraph, labels []core.BuildLabel) {
	for _, label := range labels {
		target := graph.TargetOrDie(label)
		for _, out := range target.Outputs() {
			fmt.Printf("%s\n", path.Join(target.OutDir(), out))
		}
	}
}
