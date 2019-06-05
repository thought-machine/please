package query

import (
	"bytes"
	"crypto/sha1"
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
	if !bytes.Equal(h1, h2) {
		return true
	}
	h1, err1 := sourceHash(s1, t1)
	h2, err2 := sourceHash(s2, t2)
	return !bytes.Equal(h1, h2) || err1 != nil || err2 != nil
}

// sourceHash performs a partial source hash on a target to determine if it's changed.
// This is a bit different to the one in the build package since we can't assume everything is
// necessarily present (and for performance reasons don't want to hash *everything*).
func sourceHash(state *core.BuildState, target *core.BuildTarget) (hash []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%s", r)
		}
	}()
	h := sha1.New()
	for _, tool := range target.AllTools() {
		if tool.Label() != nil {
			continue // Skip in-repo tools, that will be handled via revdeps.
		}
		for _, path := range tool.FullPaths(state.Graph) {
			result, err := state.PathHasher.Hash(path, false, false)
			if err != nil {
				return nil, err
			}
			h.Write(result)
		}
	}
	return h.Sum(nil), nil
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
