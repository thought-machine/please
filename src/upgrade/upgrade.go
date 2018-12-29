// Package upgrade handles upgrading of third-party libraries.
package upgrade

import (
	"os"
	"os/exec"
	"strings"

	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/core"
)

var log = logging.MustGetLogger("upgrade")

// labelPrefix is the prefix we use for the labels that identify how to upgrade targets.
const labelPrefix = "upgrade:"

// Upgrade runs an upgrade for a set of libraries and prints the results.
func Upgrade(state *core.BuildState, labels []core.BuildLabel) {
	for k, v := range upgrades(state, labels) {
		cmd := createCommand(k, v)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Fatalf("Failed to run %s: %s", cmd.Path, err)
		}
	}
}

func upgrades(state *core.BuildState, labels []core.BuildLabel) map[string][]string {
	m := map[string][]string{}
	for _, label := range labels {
		target := state.Graph.TargetOrDie(label)
		for _, l := range target.PrefixedLabels(labelPrefix) {
			// Labels can either be a straight build label or can prefix with the label
			// to use to identify the labels we care about, e.g.
			// upgrade:go_get:@pleasings//go/upgrade
			labels := target.Labels
			if idx := strings.IndexRune(l, ':'); idx != -1 && !core.LooksLikeABuildLabel(l) {
				labels = target.PrefixedLabels(l[:idx+1])
				l = l[idx+1:]
			}
			m[l] = append(m[l], labels...)
		}
	}
	return m
}

func createCommand(name string, args []string) *exec.Cmd {
	if core.LooksLikeABuildLabel(name) {
		executable, err := os.Executable()
		if err != nil {
			log.Fatalf("Cannot determine current executable: %s", err)
		}
		return exec.Command(executable, append([]string{"run", name, "--"}, args...)...)
	}
	return exec.Command(name, args...)
}
