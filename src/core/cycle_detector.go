package core

import (
	"fmt"
	"strings"
)

type cycleDetector struct {
	graph *BuildGraph
}

// Check runs a single check of the build graph to see if any cycles can be detected.
// If it finds one an errCycle is returned.
func (c *cycleDetector) Check() *errCycle {
	log.Debug("Running cycle detection...")
	permanent := map[*BuildTarget]struct{}{}
	temporary := map[*BuildTarget]struct{}{}

	var visit func(target *BuildTarget) ([]*BuildTarget, bool)
	visit = func(target *BuildTarget) ([]*BuildTarget, bool) {
		if _, present := permanent[target]; present {
			return nil, false
		} else if _, present := temporary[target]; present {
			return []*BuildTarget{target}, false
		}
		temporary[target] = struct{}{}
		for _, dep := range target.Dependencies() {
			if cycle, done := visit(dep); cycle != nil {
				if done || target == cycle[len(cycle)-1] {
					return cycle, true  // This target is already in the cycle
				}
				return append([]*BuildTarget{target}, cycle...), false
			}
		}
		delete(temporary, target)
		permanent[target] = struct{}{}
		return nil, false
	}

	for _, target := range c.graph.AllTargets() {
		if _, present := permanent[target]; !present {
			if cycle, _ := visit(target); cycle != nil {
				log.Debug("Cycle detection complete, cycle found: %s", cycle)
				return &errCycle{Cycle: cycle}
			}
		}
	}
	log.Debug("Cycle detection complete, no cycles found")
	return nil
}

// An errCycle is emitted when a graph cycle is detected.
type errCycle struct{
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
