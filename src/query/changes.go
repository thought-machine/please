package query

import (
	"bytes"
	"fmt"
	"os/exec"
	"sort"

	"github.com/google/shlex"

	"github.com/thought-machine/please/src/build"
	"github.com/thought-machine/please/src/core"
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
	done := make(map[*core.BuildTarget]struct{}, len(targets))
	for _, t2 := range targets {
		if t1 := before.Graph.Target(t2.Label); t1 == nil || changed(before, after, t1, t2, files) || configChanged {
			addRevdeps(after, t2, done)
		}
	}
	ret := make(core.BuildLabels, 0, len(done))
	for target := range done {
		ret = append(ret, target.Label)
	}
	sort.Sort(ret)
	return ret
}

// changed returns true if the given two targets are not equivalent.
func changed(s1, s2 *core.BuildState, t1, t2 *core.BuildTarget, files []string) bool {
	for _, f := range files {
		if t2.HasAbsoluteSource(f) {
			return true
		}
	}
	h1 := build.RuleHash(s1, t1, true, false)
	h2 := build.RuleHash(s2, t2, true, false)
	return !bytes.Equal(h1, h2)
}

// addRevdeps walks back up the reverse dependencies of a target, marking them all changed.
func addRevdeps(state *core.BuildState, target *core.BuildTarget, done map[*core.BuildTarget]struct{}) {
	if _, present := done[target]; !present && state.ShouldInclude(target) {
		done[target] = struct{}{}
		for _, revdep := range state.Graph.ReverseDependencies(target) {
			addRevdeps(state, revdep, done)
		}
	}
}
