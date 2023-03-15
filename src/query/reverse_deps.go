package query

import (
	"container/list"
	"fmt"
	"sort"

	"github.com/thought-machine/please/src/core"
)

// ReverseDeps finds all transitive targets that depend on the set of input labels.
func ReverseDeps(state *core.BuildState, labels []core.BuildLabel, level int, hidden bool) {
	targets := FindRevdeps(state, labels, hidden, true, level)
	ls := make(core.BuildLabels, 0, len(targets))

	for target := range targets {
		if state.ShouldInclude(target) {
			ls = append(ls, target.Label)
		}
	}
	sort.Sort(ls)

	for _, l := range ls {
		fmt.Println(l.String())
	}
}

// node represents a node in the build graph and the depth we visited it at.
type node struct {
	target *core.BuildTarget
	depth  int
}

// openSet represents the queue of nodes we need to process in the graph. There are no duplicates in this set and the
// queue will be ordered low to high by depth i.e. in the order they must be processed
//
// NB: We don't need to explicitly order this. Paths either cost 1 or 0, but all 0 cost paths are equivalent e.g. paths
// :lib1 -> :_lib1#foo -> :lib2, lib1 -> :_lib1#foo -> :_lib1#bar -> :lib2, and :lib1 -> lib2 all have a cost of 1 and
// will result in :lib1 and :lib2 as outputs. It doesn't matter which is explored to generate the output.
type openSet struct {
	items *list.List

	// done contains a map of targets we've already processed.
	done map[core.BuildLabel]struct{}
}

// Push implements pushing a node onto the queue of nodes to process, deduplicating nodes we've seen before.
func (os *openSet) Push(n *node) {
	if _, present := os.done[n.target.Label]; !present {
		os.done[n.target.Label] = struct{}{}
		os.items.PushBack(n)
	}
}

// Pop fetches the next node off the queue for us to process
func (os *openSet) Pop() *node {
	next := os.items.Front()
	if next == nil {
		return nil
	}
	os.items.Remove(next)

	return next.Value.(*node)
}

type revdeps struct {
	// revdeps is the map of immediate reverse dependencies
	revdeps map[core.BuildLabel][]*core.BuildTarget
	// subincludes is a map of build labels to the packages that subinclude them
	subincludes map[core.BuildLabel][]*core.Package

	// hidden is whether to count hidden targets towards the depth budget
	hidden bool

	// os is the open set of targets to process
	os *openSet

	// maxDepth is the depth budget for the search. -1 means unlimited.
	maxDepth int
}

// newRevdeps creates a new reverse dependency searcher. revdeps is non-reusable.
func newRevdeps(graph *core.BuildGraph, hidden, followSubincludes bool, maxDepth int) *revdeps {
	// Initialise a map of labels to the packages that subinclude them upfront so we can include those targets as
	// dependencies efficiently later
	subincludes := make(map[core.BuildLabel][]*core.Package)
	if followSubincludes {
		for _, pkg := range graph.PackageMap() {
			for _, inc := range pkg.Subincludes {
				subincludes[inc] = append(subincludes[inc], pkg)
			}
		}
	}

	return &revdeps{
		revdeps:           buildRevdeps(graph),
		subincludes:       subincludes,
		followSubincludes: followSubincludes,
		os: &openSet{
			items: list.New(),
			done:  map[core.BuildLabel]struct{}{},
		},
		hidden:   hidden,
		maxDepth: maxDepth,
	}
}

// buildRevdeps builds the reverse dependency map from a build graph.
func buildRevdeps(graph *core.BuildGraph) map[core.BuildLabel][]*core.BuildTarget {
	targets := graph.AllTargets()
	revdeps := make(map[core.BuildLabel][]*core.BuildTarget, len(targets))
	for _, t := range targets {
		for _, d := range t.DeclaredDependencies() {
			if t2 := graph.Target(d); t2 == nil {
				revdeps[d] = append(revdeps[d], t2)
			} else {
				for _, p := range t2.ProvideFor(t) {
					revdeps[p] = append(revdeps[p], t)
				}
			}
		}
	}
	return revdeps
}

// FindRevdeps will return a set of build targets that are reverse dependencies of the provided labels.
func FindRevdeps(state *core.BuildState, targets core.BuildLabels, hidden, followSubincludes bool, depth int) map[*core.BuildTarget]struct{} {
	r := newRevdeps(state.Graph, hidden, followSubincludes, depth)
	// Initialise the open set with the original targets
	for _, label := range targets {
		target := state.Graph.TargetOrDie(label)
		r.os.Push(&node{
			target: target,
			depth:  0,
		})
		if !hidden && !label.IsHidden() {
			for _, child := range state.Graph.PackageByLabel(label).AllTargets() {
				if child.Parent(state.Graph) == target {
					r.os.Push(&node{
						target: child,
						depth:  0,
					})
				}
			}
		}
	}
	return r.findRevdeps(state)
}

func isSameTarget(graph *core.BuildGraph, lhs, rhs *core.BuildTarget) bool {
	if lhs == rhs {
		return true
	}

	// Otherwise check they belong to the same non-hidden rule
	if lhs.Label.IsHidden() {
		lhs = lhs.Parent(graph)
	}
	if rhs.Label.IsHidden() {
		rhs = rhs.Parent(graph)
	}
	return lhs == rhs && lhs != nil
}

func (r *revdeps) findRevdeps(state *core.BuildState) map[*core.BuildTarget]struct{} {
	// 1000 is chosen pretty much arbitrarily here
	ret := make(map[*core.BuildTarget]struct{}, 1000)
	for next := r.os.Pop(); next != nil; next = r.os.Pop() {
		ts := r.revdeps[next.target.Label]
		for _, p := range r.subincludes[next.target.Label] {
			ts = append(ts, p.AllTargets()...)
		}

		for _, t := range ts {
			depth := next.depth

			// The label shouldn't count towards the depth if it's a child of the last label
			if r.hidden || !isSameTarget(state.Graph, next.target, t) {
				depth++
			}

			if next.depth < r.maxDepth || r.maxDepth == -1 {
				// This excluded  dependencies between hidden rules and their parent for when hidden is false.
				if depth > 0 {
					if r.hidden || !t.Label.IsHidden() {
						ret[t] = struct{}{}
					} else if parent := t.Parent(state.Graph); parent != nil {
						ret[parent] = struct{}{}
					}
				}

				r.os.Push(&node{
					target: t,
					depth:  depth,
				})
			}
		}
	}
	return ret
}
