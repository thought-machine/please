package query

import (
	"bytes"
	"fmt"
	"os/exec"
	"sort"

	"github.com/google/shlex"

	"build"
	"core"
)

// MustCheckout checks out the given revision.
func MustCheckout(revision, command string) {
	log.Notice("Checking out %s...", revision)
	if argv, err := shlex.Split(fmt.Sprintf(command, revision)); err != nil {
		log.Fatalf("Invalid checkout command: %s", err)
	} else if out, err := exec.Command(argv[0], argv[1:]...).CombinedOutput(); err != nil {
		log.Fatalf("Failed to check out %s: %s\n%s", revision, err, out)
	}
}

// MustGetRevision runs a command to determine the current revision.
func MustGetRevision(command string) string {
	log.Notice("Determining current revision...")
	argv, err := shlex.Split(command)
	if err != nil {
		log.Fatalf("Invalid revision command: %s", err)
	}
	out, err := exec.Command(argv[0], argv[1:]...).Output()
	if err != nil {
		log.Fatalf("Failed to determine current revision: %s\n%s", err, out)
	}
	return string(bytes.TrimSpace(out))
}

// DiffGraphs calculates the difference between two build graphs.
// Note that this is not symmetric; targets that have been removed from 'before' do not appear
// (because this is designed to be fed into 'plz test' and we can't test targets that no longer exist).
func DiffGraphs(before, after *core.BuildState, files []string) []core.BuildLabel {
	log.Notice("Calculating difference...")
	configChanged := !bytes.Equal(before.Hashes.Config, after.Hashes.Config)
	targets := after.Graph.AllTargets()
	ret := make(core.BuildLabels, 0, len(targets))
	done := make(map[*core.BuildTarget]struct{}, len(targets))
	for _, t2 := range targets {
		if t1 := before.Graph.Target(t2.Label); t1 == nil || changed(before, after, t1, t2, files) || configChanged {
			ret = append(ret, addRevdeps(after.Graph, t2, done)...)
		}
	}
	sort.Sort(ret) // reverse dependencies aren't guaranteed to be consistent
	return ret
}

// changed returns true if the given two targets are equivalent.
func changed(s1, s2 *core.BuildState, t1, t2 *core.BuildTarget, files []string) bool {
	for _, f := range files {
		if t2.HasSource(f) {
			return true
		}
	}
	h1 := build.RuleHash(s1, t1, true, false)
	h2 := build.RuleHash(s2, t2, true, false)
	return !bytes.Equal(h1, h2)
}

// addRevdeps walks back up the reverse dependencies of a target, marking them all changed.
// It returns the list of any that have, plus the target itself.
func addRevdeps(graph *core.BuildGraph, target *core.BuildTarget, done map[*core.BuildTarget]struct{}) []core.BuildLabel {
	if _, present := done[target]; present {
		return nil
	}
	done[target] = struct{}{}
	ret := []core.BuildLabel{target.Label}
	for _, revdep := range graph.ReverseDependencies(target) {
		ret = append(ret, addRevdeps(graph, revdep, done)...)
	}
	return ret
}
