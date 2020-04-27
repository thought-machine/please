package query

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"path"
	"sort"

	"github.com/thought-machine/please/src/build"
	"github.com/thought-machine/please/src/core"
)

// DiffGraphs calculates the difference between two build graphs.
// Note that this is not symmetric; targets that have been removed from 'before' do not appear
// (because this is designed to be fed into 'plz test' and we can't test targets that no longer exist).
func DiffGraphs(before, after *core.BuildState, files []string, includeDirect, includeTransitive bool) core.BuildLabels {
	log.Notice("Calculating difference...")
	changed := diffGraphs(before, after)
	log.Info("Including changed files...")
	return changedTargets(after, files, changed, includeDirect, includeTransitive)
}

// Changes calculates changes for a given set of files. It does a subset of what DiffGraphs does due to not having
// the "before" state so is less accurate (but faster).
func Changes(state *core.BuildState, files []string, includeDirect, includeTransitive bool) core.BuildLabels {
	return changedTargets(state, files, map[*core.BuildTarget]struct{}{}, includeDirect, includeTransitive)
}

// diffGraphs performs a non-recursive diff of two build graphs.
func diffGraphs(before, after *core.BuildState) map[*core.BuildTarget]struct{} {
	configChanged := !bytes.Equal(before.Hashes.Config, after.Hashes.Config)
	changed := map[*core.BuildTarget]struct{}{}
	for _, afterTarget := range after.Graph.AllTargets() {
		if beforeTarget := before.Graph.Target(afterTarget.Label); beforeTarget == nil || targetChanged(before, after, beforeTarget, afterTarget) || configChanged {
			changed[afterTarget] = struct{}{}
		}
	}
	return changed
}

// changedTargets returns the set of targets that have changed for the given files.
func changedTargets(state *core.BuildState, files []string, changed map[*core.BuildTarget]struct{}, includeDirect, includeTransitive bool) core.BuildLabels {
	for _, filename := range files {
		for dir := filename; dir != "." && dir != "/"; {
			dir = path.Dir(dir)
			if pkg := state.Graph.Package(dir, ""); pkg != nil {
				// This is the package closest to the file; it is the only one allowed to consume it directly.
				for _, t := range pkg.AllTargets() {
					if t.HasAbsoluteSource(filename) {
						changed[t] = struct{}{}
					}
				}
				break
			}
		}
	}
	if includeDirect || includeTransitive {
		changed2 := make(map[*core.BuildTarget]struct{}, len(changed))
		for target := range changed {
			addRevdeps(state, changed2, target, includeDirect, includeTransitive)
		}
		changed = changed2
	}
	labels := make(core.BuildLabels, 0, len(changed))
	for target := range changed {
		if state.ShouldInclude(target) {
			labels = append(labels, target.Label)
		}
	}
	sort.Sort(labels)
	return labels
}

// targetChanged returns true if the given two targets are not equivalent.
func targetChanged(s1, s2 *core.BuildState, t1, t2 *core.BuildTarget) bool {
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
func addRevdeps(state *core.BuildState, changed map[*core.BuildTarget]struct{}, target *core.BuildTarget, includeDirect, includeTransitive bool) {
	if _, present := changed[target]; !present {
		if state.ShouldInclude(target) {
			changed[target] = struct{}{}
		}
		if includeDirect || includeTransitive {
			for _, revdep := range state.Graph.ReverseDependencies(target) {
				addRevdeps(state, changed, revdep, false, includeTransitive)
			}
		}
	}
}
