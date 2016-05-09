package query

import (
	"fmt"
	"strings"

	"core"
)

func QueryAllTargets(graph *core.BuildGraph, labels core.BuildLabels, showHidden bool) {
	for _, label := range labels {
		if showHidden || !strings.HasPrefix(label.Name, "_") {
			fmt.Printf("%s\n", label)
		}
	}
}
