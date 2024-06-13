package query

import (
	"fmt"
	"slices"

	"github.com/thought-machine/please/src/core"
)

// SomePath finds and returns a path between two targets, or between one and a set of targets.
// Useful for a "why on earth do I depend on this thing" type query.
func SomePath(graph *core.BuildGraph, from, to, except []core.BuildLabel, showHidden, ffDefaultProvide bool) error {
	s := somepath{
		graph:            graph,
		except:           make(map[core.BuildLabel]struct{}, len(except)),
		memo:             map[core.BuildLabel]map[core.BuildLabel]struct{}{},
		ffDefaultProvide: ffDefaultProvide,
	}
	for _, ex := range except {
		s.except[ex] = struct{}{}
	}
	for _, l1 := range expandAllTargets(graph, from) {
		for _, l2 := range expandAllTargets(graph, to) {
			if path := s.SomePath(l1, l2); len(path) != 0 {
				fmt.Println("Found path:")
				if !showHidden {
					// Filter path to just non-hidden targets
					for i, x := range path {
						path[i] = x.Parent()
					}
					path = slices.Compact(path)
				}
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
	graph            *core.BuildGraph
	memo             map[core.BuildLabel]map[core.BuildLabel]struct{}
	except           map[core.BuildLabel]struct{}
	ffDefaultProvide bool
}

func (s *somepath) SomePath(target1, target2 core.BuildLabel) []core.BuildLabel {
	// Have to try this both ways around since we don't know which is a dependency of the other.
	if path := s.somePath(target1, target2); len(path) != 0 {
		return path
	}
	return s.somePath(target2, target1)
}

func (s *somepath) somePath(target1, target2 core.BuildLabel) []core.BuildLabel {
	m, present := s.memo[target2]
	if !present {
		m = map[core.BuildLabel]struct{}{}
		s.memo[target2] = m
	}
	return somePath(s.graph, s.graph.TargetOrDie(target1), s.graph.TargetOrDie(target2), m, s.except, s.ffDefaultProvide)
}

func somePath(graph *core.BuildGraph, target1, target2 *core.BuildTarget, seen, except map[core.BuildLabel]struct{}, ffDefaultProvide bool) []core.BuildLabel {
	if target1.Label == target2.Label {
		return []core.BuildLabel{target1.Label}
	} else if target1.Parent(graph) == target2 {
		// If there's some path to the parent of the named target, count that. This is usually what you want e.g. in the
		// case of protos where the named target is just a filegroup that isn't actually depended on after the
		// require/provide is resolved.
		return []core.BuildLabel{target1.Label}
	} else if _, present := seen[target1.Label]; present {
		return nil
	}
	seen[target1.Label] = struct{}{}
	for _, dep := range target1.DeclaredDependencies() {
		if t := graph.Target(dep); t != nil {
			if _, present := except[t.Label]; present {
				continue
			}
			for _, l := range t.ProvideFor(target1, ffDefaultProvide) {
				if path := somePath(graph, graph.TargetOrDie(l), target2, seen, except, ffDefaultProvide); len(path) != 0 {
					return append([]core.BuildLabel{target1.Label}, path...)
				}
			}
		}
	}
	if target1.Subrepo != nil && target1.Subrepo.Target != nil {
		if path := somePath(graph, target1.Subrepo.Target, target2, seen, except, ffDefaultProvide); len(path) != 0 {
			return append([]core.BuildLabel{target1.Label}, path...)
		}
	}
	return nil
}
