package query

import "fmt"
import "path"
import "core"

// QueryTargetOutputs prints all output files for a set of targets.
func QueryTargetOutputs(graph *core.BuildGraph, labels []core.BuildLabel) {
	for _, label := range labels {
		target := graph.TargetOrDie(label)
		for _, out := range target.Outputs() {
			fmt.Printf("%s\n", path.Join(target.OutDir(), out))
		}
	}
}
