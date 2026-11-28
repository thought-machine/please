package query

import (
	"bufio"
	"os"
	"strings"

	"github.com/thought-machine/please/src/core"
)

// AllTargets simply prints all the targets according to some expression.
func AllTargets(graph *core.BuildGraph, labels core.BuildLabels, showHidden bool) {
	// This can print a very large number of targets, so buffer it rather than doing a syscall for each one.
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()
	for _, label := range labels {
		if showHidden || !strings.HasPrefix(label.Name, "_") {
			w.WriteString(label.String())
			w.WriteByte('\n')
		}
	}
}
