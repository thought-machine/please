package query

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/thought-machine/please/src/core"
)

// TargetOutputs prints all output files for a set of targets.
func TargetOutputs(graph *core.BuildGraph, labels []core.BuildLabel, useJSON bool) {
	if useJSON {
		targetOutputsJSON(graph, labels)
	} else {
		targetOutputsFlat(graph, labels)
	}
}

func targetOutputsFlat(graph *core.BuildGraph, labels []core.BuildLabel) {
	for _, label := range labels {
		target := graph.TargetOrDie(label)
		for _, out := range target.Outputs() {
			fmt.Printf("%s\n", filepath.Join(target.OutDir(), out))
		}
	}
}

func targetOutputsJSON(graph *core.BuildGraph, labels []core.BuildLabel) {
	data := map[string][]string{}
	for _, label := range labels {
		target := graph.TargetOrDie(label)
		for _, out := range target.Outputs() {
			data[label.String()] = append(data[label.String()], filepath.Join(target.OutDir(), out))
		}
	}
	bs, err := json.Marshal(data)
	if err != nil {
		log.Fatalf("failed to marshal JSON: %v", err)
	}
	if _, err := os.Stdout.Write(bs); err != nil {
		log.Fatalf("failed to write JSON: %v", err)
	}
}
