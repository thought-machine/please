package query

import (
	"bytes"
	"path/filepath"
	"slices"

	"github.com/thought-machine/please/src/build"
	"github.com/thought-machine/please/src/core"
)

// DiffGraphs calculates the difference between two build graphs.
// Note that this is not symmetric; targets that have been removed from 'before' do not appear
// (because this is designed to be fed into 'plz test' and we can't test targets that no longer exist).
func DiffGraphs(before, after *core.BuildState, files []string, level int, includeSubrepos bool) core.BuildLabels {
	return DiffSnapshot(NewSnapshot(before), after, files, level, includeSubrepos)
}

// DiffSnapshot is like DiffGraphs but compares against a snapshot of the "before" graph.
func DiffSnapshot(before *GraphSnapshot, after *core.BuildState, files []string, level int, includeSubrepos bool) core.BuildLabels {
	log.Notice("Calculating difference...")
	changed := diffSnapshot(before, after)
	log.Debugf("Number of changed targets on a non-recursive diff between before and after build graphs: %d", len(changed))

	log.Info("Including changed files...")
	return changedTargets(after, files, changed, level, includeSubrepos)
}

// Changes calculates changes for a given set of files. It does a subset of what DiffGraphs does due to not having
// the "before" state so is less accurate (but faster).
func Changes(state *core.BuildState, files []string, level int, includeSubrepos bool) core.BuildLabels {
	return changedTargets(state, files, map[*core.BuildTarget]struct{}{}, level, includeSubrepos)
}

// diffSnapshot performs a non-recursive diff of a build graph against a snapshot of a previous one.
func diffSnapshot(before *GraphSnapshot, after *core.BuildState) map[*core.BuildTarget]struct{} {
	configChanged := !bytes.Equal(before.Header.ConfigHash, after.Hashes.Config)
	log.Debugf("Has config changed between before and after build states: %v", configChanged)

	changed := map[*core.BuildTarget]struct{}{}
	for _, t := range after.Graph.AllTargets() {
		if b, present := before.Targets[t.Label.String()]; !present || configChanged ||
			!bytes.Equal(b.Rule, build.RuleHash(after, t, true, false)) ||
			!bytes.Equal(b.Tools, toolsHash(after, t)) {
			changed[t] = struct{}{}
		}
	}
	return changed
}

// changedTargets returns the set of targets that have changed for the given files.
func changedTargets(state *core.BuildState, files []string, changed map[*core.BuildTarget]struct{}, level int, includeSubrepos bool) core.BuildLabels {
	for _, filename := range files {
		for dir := filename; dir != "." && dir != "/"; {
			dir = filepath.Dir(dir)
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
		revdeps := FindRevdeps(state, labels, true, false, includeSubrepos, level)
		for dep := range revdeps {
			if _, present := changed[dep]; !present {
				labels = append(labels, dep.Label)
			}
		}
	}

	ls := make(core.BuildLabels, 0, len(labels))
	for _, l := range labels {
		t := state.Graph.TargetOrDie(l)
		if state.ShouldInclude(t) && (includeSubrepos || t.Subrepo == nil) {
			ls = append(ls, l)
		}
	}
	slices.SortFunc(ls, core.BuildLabel.Compare)
	return ls
}
