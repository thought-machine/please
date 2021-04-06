package query

import (
	"fmt"

	"github.com/thought-machine/please/src/core"
)

// SomePath finds and returns a path between two targets, or between one and a set of targets.
// Useful for a "why on earth do I depend on this thing" type query.
func SomePath(graph *core.BuildGraph, from, to []core.BuildLabel) error {
	s := somepath{
		memo: map[core.BuildLabel]map[core.BuildLabel]struct{}{},
	}
	for _, l1 := range expandAllTargets(graph, from) {
		for _, l2 := range expandAllTargets(graph, to) {
			if path := s.SomePath(graph.TargetOrDie(l1), graph.TargetOrDie(l2)); len(path) != 0 {
				fmt.Println("Found path:")
				for _, l := range path {
					fmt.Printf("  %s\n", l)
				}
				return nil
			}
		}
	}
	if len(from) == 1 && len(to) == 1 {
		return fmt.Errorf("Couldn't find any dependency path between %s and %s", from[0], to[0])
	}
	return fmt.Errorf("Couldn't find any dependency path between those targets")
}

// expandAllTargets expands any :all labels in the given set.
func expandAllTargets(graph *core.BuildGraph, labels []core.BuildLabel) []core.BuildLabel {
	ret := make([]core.BuildLabel, 0, len(labels))
	for _, l := range labels {
		if l.IsAllTargets() {
			for _, t := range graph.PackageOrDie(l).AllTargets() {
				ret = append(ret, t.Label)
			}
		} else {
			ret = append(ret, l)
		}
	}
	return ret
}

type somepath struct {
	memo map[core.BuildLabel]map[core.BuildLabel]struct{}
}

func (s *somepath) SomePath(target1, target2 *core.BuildTarget) []core.BuildLabel {
	// Have to try this both ways around since we don't know which is a dependency of the other.
	if path := s.somePath(target1, target2); len(path) != 0 {
		return path
	}
	return s.somePath(target2, target1)
}

func (s *somepath) somePath(target1, target2 *core.BuildTarget) []core.BuildLabel {
	m, present := s.memo[target2.Label]
	if !present {
		m = map[core.BuildLabel]struct{}{}
		s.memo[target2.Label] = m
	}
	return somePath(target1, target2, m)
}

func somePath(target1, target2 *core.BuildTarget, seen map[core.BuildLabel]struct{}) []core.BuildLabel {
	if target1.Label == target2.Label {
		return []core.BuildLabel{target1.Label}
	} else if _, present := seen[target1.Label]; present {
		return nil
	}
	for _, dep := range target1.Dependencies() {
		if path := somePath(dep, target2, seen); len(path) != 0 {
			return append([]core.BuildLabel{target1.Label}, path...)
		}
	}
	seen[target1.Label] = struct{}{}
	return nil
}

// filterPath filters out any internal targets on a path between two targets.
func filterPath(path []core.BuildLabel) []core.BuildLabel {
	ret := []core.BuildLabel{path[0]}
	last := path[0]
	for _, l := range path {
		if l.Parent() != last {
			ret = append(ret, l)
			last = l
		}
	}
	return ret
}
