package plz

import (
	"bytes"
	"context"
	"fmt"
	"iter"
	"runtime/pprof"
	"slices"
	"strings"
	"time"

	"github.com/thought-machine/please/src/core"
)

// cycleCheckDuration is the length of time we allow inactivity for before we trigger cycle detection.
const cycleCheckDuration = 5 * time.Second

type cycleDetector struct {
	graph *core.BuildGraph
}

// Check runs a single check of the build graph to see if any cycles can be detected.
// If it finds one an errCycle is returned.
//
// This operates on labels rather than targets; many of the things we need to consider don't have a
// target at all (packages), or don't have one yet (anything that is still being parsed), and those
// are exactly the ones that are interesting when we're stuck.
func (c *cycleDetector) Check() *errCycle {
	log.Debug("Running cycle detection...")
	complete := map[core.BuildLabel]struct{}{}
	partial := map[core.BuildLabel]struct{}{}

	// visit visits a label and all its transitive dependencies. As each is visited they are marked as
	// partially visited; when we bottom out a tree successfully we mark it as completely visited (this
	// saves us from revisiting any node we've successfully visited before).
	// If a cycle is found it returns a slice of the labels in that cycle, and a bool indicating if the
	// cycle is complete or not (if not the caller will need to add its node to it as well).
	var visit func(label core.BuildLabel) ([]core.BuildLabel, bool)
	visit = func(label core.BuildLabel) ([]core.BuildLabel, bool) {
		if _, present := complete[label]; present {
			return nil, false
		} else if _, present := partial[label]; present {
			return []core.BuildLabel{label}, false
		}
		partial[label] = struct{}{}
		for dep := range c.deps(label) {
			if cycle, done := visit(dep); cycle != nil {
				if done || label == cycle[len(cycle)-1] {
					return cycle, true // This label is already in the cycle
				}
				return append([]core.BuildLabel{label}, cycle...), false
			}
		}
		delete(partial, label)
		complete[label] = struct{}{}
		return nil, false
	}

	for label := range c.roots() {
		if _, present := complete[label]; !present {
			if cycle, _ := visit(label); cycle != nil {
				log.Debug("Cycle detection complete, cycle found: %s", cycle)
				return &errCycle{Cycle: cycle}
			}
		}
	}
	log.Debug("Cycle detection complete, no cycles found")
	// Dump the goroutine info in case that helps to shed light
	var buf bytes.Buffer
	pprof.Lookup("goroutine").WriteTo(&buf, 1)
	log.Debug("Current stacks: %s", buf.String())
	return nil
}

// roots returns the labels we start walking from, which is all targets and anything that's subincluded something else.
func (c *cycleDetector) roots() iter.Seq[core.BuildLabel] {
	return func(yield func(core.BuildLabel) bool) {
		for _, t := range c.graph.AllTargets() {
			if !yield(t.Label) {
				return
			}
		}
		for l := range c.graph.SubincludeNodes() {
			if !yield(l) {
				return
			}
		}
	}
}

// deps returns everything that the given label has to wait for.
func (c *cycleDetector) deps(label core.BuildLabel) iter.Seq[core.BuildLabel] {
	if label.IsAllTargets() {
		return c.graph.Subincludes(label)
	}
	target := c.graph.Target(label)
	if target == nil {
		// We don't have this target yet, so we must be waiting for its package to parse and define it.
		return slices.Values([]core.BuildLabel{{
			Subrepo:     label.Subrepo,
			PackageName: label.PackageName,
			Name:        "all",
		}})
	}
	// Snapshot the dependencies rather than holding the target's lock for the whole recursive walk.
	deps := slices.Collect(target.DeclaredDependencies())
	return slices.Values(append(deps, slices.Collect(c.graph.Subincludes(label))...))
}

// An errCycle is emitted when a graph cycle is detected.
type errCycle struct {
	Cycle []core.BuildLabel
}

func (err *errCycle) Error() string {
	labels := make([]string, len(err.Cycle)+1)
	for i, l := range err.Cycle {
		labels[i] = describeCycleLabel(l)
	}
	labels[len(labels)-1] = labels[0]
	return fmt.Sprintf("Dependency cycle found:\n%s\nSorry, but you'll have to refactor your build files to avoid this cycle", strings.Join(labels, "\n -> "))
}

// describeCycleLabel returns the description of a label in a cycle; packages get called out as such
// since otherwise it's not obvious why a pseudo-label has appeared in the middle of one.
func describeCycleLabel(label core.BuildLabel) string {
	if label.IsAllTargets() {
		return "parse of " + strings.TrimSuffix(label.String(), ":all")
	}
	return label.String()
}

// checkForCycles consumes a stream of build results and triggers cycle detection when appropriate
func checkForCycles(state *core.BuildState, results <-chan *core.BuildResult, cancel context.CancelCauseFunc) {
	checker := cycleDetector{graph: state.Graph}
	active := map[*core.BuildTarget]struct{}{}
	t := time.NewTimer(cycleCheckDuration)
	defer t.Stop()
	for {
		select {
		case result, ok := <-results:
			if !ok {
				return // results channel closed means the build is complete
			}
			t.Reset(cycleCheckDuration)
			if target := result.Target; target != nil {
				if result.Status.IsActive() {
					active[target] = struct{}{}
				} else {
					delete(active, target)
				}
			}
		case <-t.C:
			t.Reset(cycleCheckDuration)
			if len(active) > 0 {
				continue
			}
			go func() {
				if err := checker.Check(); err != nil {
					state.LogBuildError(err.Cycle[0], core.TargetBuildFailed, err, "")
					cancel(err)
				}
			}()
		}
	}
}
