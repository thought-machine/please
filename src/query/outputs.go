package query

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"github.com/thought-machine/please/src/core"
)

// TargetOutputs prints all output files for a set of targets.
func TargetOutputs(out io.Writer, graph *core.BuildGraph, labels []core.BuildLabel, useJSON bool) {
	if useJSON {
		targetOutputsJSON(out, graph, labels)
	} else {
		targetOutputsFlat(out, graph, labels)
	}
}

func targetOutputsFlat(out io.Writer, graph *core.BuildGraph, labels []core.BuildLabel) {
	for _, label := range labels {
		target := graph.TargetOrDie(label)
		for _, o := range target.Outputs() {
			fmt.Fprintf(out, "%s\n", filepath.Join(target.OutDir(), o))
		}
	}
}

func targetOutputsJSON(out io.Writer, graph *core.BuildGraph, labels []core.BuildLabel) {
	data := map[string][]string{}
	for _, label := range labels {
		target := graph.TargetOrDie(label)
		for _, o := range target.Outputs() {
			data[label.String()] = append(data[label.String()], filepath.Join(target.OutDir(), o))
		}
	}
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		log.Fatalf("failed to write JSON: %v", err)
	}
}
