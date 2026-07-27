package plz

import (
	"bytes"
	"context"
	"fmt"
	"runtime/pprof"
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
func (c *cycleDetector) Check() *errCycle {
	log.Debug("Running cycle detection...")
	complete := map[*core.BuildTarget]struct{}{}
	partial := map[*core.BuildTarget]struct{}{}

	// visit visits a target and all its transitive dependencies. As each is visited they are marked as
	// partially visited; when we bottom out a tree successfully we mark it as completely visited (this
	// saves us from revisiting any node we've successfully visited before).
	// If a cycle is found it returns a slice of the targets in that cycle, and a bool indicating if the
	// cycle is complete or not (if not the caller will need to add its node to it as well).
	var visit func(target *core.BuildTarget) ([]*core.BuildTarget, bool)
	visit = func(target *core.BuildTarget) ([]*core.BuildTarget, bool) {
		if _, present := complete[target]; present {
			return nil, false
		} else if _, present := partial[target]; present {
			return []*core.BuildTarget{target}, false
		}
		partial[target] = struct{}{}
		// Ignore anything we can't resolve; we run while the build is still going on so it's
		// entirely normal for parts of the graph not to exist yet.
		deps, _ := target.Dependencies(c.graph)
		for _, dep := range deps {
			if cycle, done := visit(dep); cycle != nil {
				if done || target == cycle[len(cycle)-1] {
					return cycle, true // This target is already in the cycle
				}
				return append([]*core.BuildTarget{target}, cycle...), false
			}
		}
		delete(partial, target)
		complete[target] = struct{}{}
		return nil, false
	}

	for _, target := range c.graph.AllTargets() {
		if _, present := complete[target]; !present {
			if cycle, _ := visit(target); cycle != nil {
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

// An errCycle is emitted when a graph cycle is detected.
type errCycle struct {
	Cycle []*core.BuildTarget
}

func (err *errCycle) Error() string {
	labels := make([]string, len(err.Cycle)+1)
	for i, t := range err.Cycle {
		labels[i] = t.Label.String()
	}
	labels[len(labels)-1] = labels[0]
	return fmt.Sprintf("Dependency cycle found:\n%s\nSorry, but you'll have to refactor your build files to avoid this cycle", strings.Join(labels, "\n -> "))
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
					state.LogBuildError(err.Cycle[0].Label, core.TargetBuildFailed, err, "")
					cancel(err)
				}
			}()
		}
	}
}
