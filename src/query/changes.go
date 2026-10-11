package query

import (
	"bytes"
	"crypto/sha1"
	"path/filepath"
	"runtime"
	"sort"

	"golang.org/x/sync/errgroup"

	"github.com/thought-machine/please/src/build"
	"github.com/thought-machine/please/src/core"
)

// DiffGraphs calculates the difference between two build graphs.
// Note that this is not symmetric; targets that have been removed from 'before' do not appear
// (because this is designed to be fed into 'plz test' and we can't test targets that no longer exist).
func DiffGraphs(before *GraphSummary, after *core.BuildState, files []string, level int, includeSubrepos bool) core.BuildLabels {
	log.Notice("Calculating difference...")
	changed := diffGraphs(before, after)
	log.Debugf("Number of changed targets on a non-recursive diff between before and after build graphs: %d", len(changed))

	log.Info("Including changed files...")
	return changedTargets(after, files, changed, level, includeSubrepos)
}

// Changes calculates changes for a given set of files. It does a subset of what DiffGraphs does due to not having
// the "before" state so is less accurate (but faster).
func Changes(state *core.BuildState, files []string, level int, includeSubrepos bool) core.BuildLabels {
	return changedTargets(state, files, map[*core.BuildTarget]struct{}{}, level, includeSubrepos)
}

// diffGraphs performs a non-recursive diff of two build graphs.
func diffGraphs(before *GraphSummary, after *core.BuildState) map[*core.BuildTarget]struct{} {
	changed := map[*core.BuildTarget]struct{}{}
	// Shortcut everything if these aren't the same
	if !bytes.Equal(before.configHash, after.Hashes.Config) {
		log.Debug("Config has changed, emitting every target")
		for t := range after.Graph.IterTargets() {
			changed[t] = struct{}{}
		}
		return changed
	}

	ch := make(chan *core.BuildTarget, 1000)
	var g errgroup.Group
	g.SetLimit(runtime.NumCPU())
	for afterTarget := range after.Graph.IterTargets() {
		g.Go(func() error {
			if beforeHash, present := before.targets[afterTarget.Label]; !present || beforeHash != targetHash(after, afterTarget) {
				ch <- afterTarget
			}
			return nil
		})
	}
	go func() {
		g.Wait()
		close(ch)
	}()
	for t := range ch {
		changed[t] = struct{}{}
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
	sort.Sort(ls)
	return ls
}

// targetHash returns a combined hash of the target which we use to check if they have changed.
func targetHash(state *core.BuildState, target *core.BuildTarget) (ret [sha1.Size]byte) {
	h := sha1.New()
	// TODO(peterebden): We could slightly improve this by having an interface to pass our own hasher in instead of hashing this twice.
	h.Write(build.RuleHash(state, target, true, false))
	// The only thing we need to check source-wise is how tool paths resolve outside the repo.
	// Changes to any tools within the repo are caught here anyway.
	for _, tool := range target.AllTools() {
		if _, ok := tool.Label(); !ok {
			h.Write([]byte(tool.String()))
		}
	}
	h.Sum(ret[:0])
	return ret
}

// Summarise calculates a summary of a build state that is sufficient for change calculation, allowing the caller to free the 'before' state.
func Summarise(state *core.BuildState) *GraphSummary {
	m := map[core.BuildLabel][sha1.Size]byte{}
	// TODO(peterebden): Would we benefit from a graph member to tell us how big this is?
	for t := range state.Graph.IterTargets() {
		m[t.Label] = targetHash(state, t)
	}
	return &GraphSummary{
		configHash: state.Hashes.Config,
		targets:    m,
	}
}

// A GraphSummary holds a summary of a build graph that is sufficient for change calculation
type GraphSummary struct {
	configHash []byte
	targets    map[core.BuildLabel][sha1.Size]byte
}
