package core

import (
	"fmt"
	"strings"
)

type cycleDetector struct {
	graph   *BuildGraph
	stopped bool
}

// Check runs a single check of the build graph to see if any cycles can be detected.
// If it finds one an errCycle is returned.
func (c *cycleDetector) Check() *errCycle {
	if c.stopped {
		return nil
	}
	log.Debug("Running cycle detection...")
	complete := map[*BuildTarget]struct{}{}
	partial := map[*BuildTarget]struct{}{}

	// visit visits a target and all its transitive dependencies. As each is visited they are marked as
	// partially visited; when we bottom out a tree successfully we mark it as completely visited (this
	// saves us from revisiting any node we've successfully visited before).
	// If a cycle is found it returns a slice of the targets in that cycle, and a bool indicating if the
	// cycle is complete or not (if not the caller will need to add its node to it as well).
	var visit func(target *BuildTarget) ([]*BuildTarget, bool)
	visit = func(target *BuildTarget) ([]*BuildTarget, bool) {
		if c.stopped {
			return nil, false
		} else if _, present := complete[target]; present {
			return nil, false
		} else if _, present := partial[target]; present {
			return []*BuildTarget{target}, false
		}
		partial[target] = struct{}{}
		for _, dep := range target.Dependencies() {
			if cycle, done := visit(dep); cycle != nil {
				if done || target == cycle[len(cycle)-1] {
					return cycle, true // This target is already in the cycle
				}
				return append([]*BuildTarget{target}, cycle...), false
			}
		}
		delete(partial, target)
		complete[target] = struct{}{}
		return nil, false
	}

	for _, target := range c.graph.AllTargets() {
		if c.stopped {
			log.Debug("Cycle detection terminated")
			return nil
		}
		if _, present := complete[target]; !present {
			if cycle, _ := visit(target); cycle != nil {
				log.Debug("Cycle detection complete, cycle found: %s", cycle)
				return &errCycle{Cycle: cycle}
			}
		}
	}
	log.Debug("Cycle detection complete, no cycles found")
	return nil
}

// Stop stops any existing run of the cycle detector.
func (c *cycleDetector) Stop() {
	c.stopped = true
}

// An errCycle is emitted when a graph cycle is detected.
type errCycle struct {
	Cycle []*BuildTarget
}

func (err *errCycle) Error() string {
	labels := make([]string, len(err.Cycle))
	for i, t := range err.Cycle {
		labels[i] = t.Label.String()
	}
	labels = append(labels, labels[0])
	return fmt.Sprintf("Dependency cycle found:\n%s\nSorry, but you'll have to refactor your build files to avoid this cycle", strings.Join(labels, "\n -> "))
}
