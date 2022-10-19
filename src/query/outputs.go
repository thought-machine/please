package query

import (
	"fmt"
	"path/filepath"

	"github.com/thought-machine/please/src/core"
)

// TargetOutputs prints all output files for a set of targets.
func TargetOutputs(graph *core.BuildGraph, labels []core.BuildLabel) {
	for _, label := range labels {
		target := graph.TargetOrDie(label)
		for _, out := range target.Outputs() {
			fmt.Printf("%s\n", filepath.Join(target.OutDir(), out))
		}
	}
}
