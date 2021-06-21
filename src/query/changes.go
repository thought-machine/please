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
func DiffGraphs(before, after *core.BuildState, files []string, level int) core.BuildLabels {
	log.Notice("Calculating difference...")
	changed := diffGraphs(before, after)
	log.Info("Including changed files...")
	return changedTargets(after, files, changed, level)
}

// Changes calculates changes for a given set of files. It does a subset of what DiffGraphs does due to not having
// the "before" state so is less accurate (but faster).
func Changes(state *core.BuildState, files []string, level int) core.BuildLabels {
	return changedTargets(state, files, map[*core.BuildTarget]struct{}{}, level)
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
func changedTargets(state *core.BuildState, files []string, changed map[*core.BuildTarget]struct{}, level int) core.BuildLabels {
	for _, filename := range files {
		for dir := filename; dir != "." && dir != "/"; {
			dir = path.Dir(dir)
			pkgName := dir
			if pkgName == "." {
				pkgName = ""
			}
			if pkg := state.Graph.Package(pkgName, ""); pkg != nil {
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
	labels := make(core.BuildLabels, 0, len(changed))
	for target := range changed {
		labels = append(labels, target.Label)
	}

	if level != 0 {
		revdeps := FindRevdeps(state, labels, true, false, level)
		for dep := range revdeps {
			if _, present := changed[dep]; !present {
				labels = append(labels, dep.Label)
			}
		}
	}

	ls := make(core.BuildLabels, 0, len(labels))
	for _, l := range labels {
		if state.ShouldInclude(state.Graph.TargetOrDie(l)) {
			ls = append(ls, l)
		}
	}
	sort.Sort(ls)
	return ls
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
		if _, ok := tool.Label(); ok {
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
