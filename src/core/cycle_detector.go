package core

import (
	"fmt"
	"strings"
)

type cycleDetector struct {
	state *BuildState
}

// Check runs a single check of the build graph to see if any cycles can be detected.
// If it finds one it logs an async error to the state object.
func (c *cycleDetector) Check() error {
	permanent := map[*BuildTarget]struct{}{}
	temporary := map[*BuildTarget]struct{}{}

	var visit func(target *BuildTarget) []*BuildTarget
	visit = func(target *BuildTarget) []*BuildTarget {
		if _, present := permanent[target]; present {
			return nil
		} else if _, present := temporary[target]; present {
			return []*BuildTarget{target}
		}
		temporary[target] = struct{}{}
		for _, dep := range target.Dependencies() {
			if cycle := visit(dep); cycle != nil {
				return append([]*BuildTarget{target}, cycle...)
			}
		}
		delete(temporary, target)
		permanent[target] = struct{}{}
		return nil
	}

	for _, target := range c.state.Graph.AllTargets() {
		if _, present := permanent[target]; !present {
			if cycle := visit(target); cycle != nil {
				err := errCycle{Cycle: cycle}
				c.state.LogBuildError(0, cycle[0].Label, TargetBuildFailed, err, "")
				c.state.Stop()
				return err
			}
		}
	}
	return nil
}

// An errCycle is emitted when a graph cycle is detected.
type errCycle struct{
	Cycle []*BuildTarget
}

func (err errCycle) Error() string {
	labels := make([]string, len(err.Cycle))
	for i, t := range err.Cycle {
		labels[i] = t.Label.String()
	}
	return fmt.Sprintf("Dependency cycle found:\n%s\nSorry, but you'll have to refactor your build files to avoid this cycle", strings.Join(labels, "\n -> "))
}
