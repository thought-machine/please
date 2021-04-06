package query

import (
	"container/list"
	"fmt"
	"sort"

	"github.com/thought-machine/please/src/core"
)

// ReverseDeps finds all transitive targets that depend on the set of input labels.
func ReverseDeps(state *core.BuildState, labels []core.BuildLabel, level int, hidden bool) {
	targets := FindRevdeps(state, labels, hidden, level)
	ls := make(core.BuildLabels, 0, len(targets))

	done := make(map[core.BuildLabel]struct{}, len(targets))
	for target := range targets {
		if state.ShouldInclude(target) {
			label := target.Label
			// Resolve targets to their parent unless including hidden targets
			if parent := target.Parent(state.Graph); !hidden && parent != nil {
				label = parent.Label
			}

			// Exclude duplicates where we had many children of the same target
			if _, present := done[label]; present {
				continue
			}
			done[label] = struct{}{}
			ls = append(ls, label)
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

// openSet represents the queue of nodes we need to process in the graph
type openSet struct {
	items *list.List

	// depths contains a map of labels and the depth we processed them at.
	// This is used to efficiently push nodes to the queue when we revisit them
	// at a lower depth
	depths map[core.BuildLabel]int
}

// Push implements pushing a node onto the queue of nodes to process. It will only add the node if we need to to process
// it because either it 1) hasn't been seen before, or 2) has been seen but at a lower level
func (os *openSet) Push(n *node) {
	// Add the node to the open set of nodes to process if we've not seen it before or we saw it as a deeper level
	// and need to reprocess
	if depth, present := os.depths[n.target.Label]; !present || depth > n.depth {
		os.depths[n.target.Label] = n.depth
		os.items.PushBack(n)
	}
}

func (os *openSet) Pop() *node {
	next := os.items.Front()
	if next == nil {
		return nil
	}
	os.items.Remove(next)

	return next.Value.(*node)
}

type revdeps struct {
	// subincludes is a map of build labels to the packages that subinclude them
	subincludes map[core.BuildLabel][]*core.Package

	// os is the open set of targets to process
	os *openSet

	// hidden is whether to count hidden targets towards the depth budget
	hidden   bool

	// maxDepth is the depth budget for the search. -1 means unlimited.
	maxDepth int
}

// newRevdeps creates a new reverse dependency searcher. revdeps is non-reusable.
func newRevdeps(graph *core.BuildGraph, hidden bool, maxDepth int) *revdeps {
	// Initialise a map of labels to the packages that subinclude them upfront so we can include those targets as
	// dependencies efficiently later
	subincludes := make(map[core.BuildLabel][]*core.Package)
	for _, pkg := range graph.PackageMap() {
		for _, inc := range pkg.Subincludes {
			subincludes[inc] = append(subincludes[inc], pkg)
		}
	}

	return &revdeps{
		subincludes: subincludes,
		os: &openSet{
			items:  list.New(),
			depths: map[core.BuildLabel]int{},
		},
		hidden:   hidden,
		maxDepth: maxDepth,
	}
}

// FindRevdeps will return a list of build targets that are reverse dependencies of the provided labels. This may
// include duplicate targets.
func FindRevdeps(state *core.BuildState, targets core.BuildLabels, hidden bool, depth int) map[*core.BuildTarget]struct{} {
	r := newRevdeps(state.Graph, hidden, depth)
	// Initialise the open set with the original targets
	for _, t := range targets {
		r.os.Push(&node{
			target: state.Graph.TargetOrDie(t),
			depth:  0,
		})
	}
	return r.findRevdeps(state)
}

func (r *revdeps) findRevdeps(state *core.BuildState) map[*core.BuildTarget]struct{} {
	// 1000 is chosen pretty much arbitrarily here
	ret := make(map[*core.BuildTarget]struct{}, 1000)
	for next := r.os.Pop(); next != nil; next = r.os.Pop() {
		ts := state.Graph.ReverseDependencies(next.target)

		for _, p := range r.subincludes[next.target.Label] {
			ts = append(ts, p.AllTargets()...)
		}

		for _, t := range ts {
			depth := next.depth
			parent := t.Parent(state.Graph)

			// The label shouldn't count towards the depth if it's a child of the last label
			if r.hidden || parent == nil || parent != next.target {
				depth++
			}

			// We can skip adding to the open set if the depth of the next non-child label pushes us over the budget
			// but we must make sure to add child labels at the current depth.
			if (next.depth+1) <= r.maxDepth || r.maxDepth == -1 {
				if r.hidden || !t.Label.IsHidden() {
					ret[t] = struct{}{}
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
