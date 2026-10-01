// Representation of the build graph.
// The graph of build targets forms a DAG which we discover from the top
// down and then build bottom-up.

package core

import (
	"iter"
	"slices"
	"sort"

	"github.com/thought-machine/please/src/cmap"
)

// A BuildGraph contains all the loaded targets and packages and maintains their
// relationships, especially reverse dependencies which are calculated here.
type BuildGraph struct {
	// Map of all currently known targets by their label.
	targets *cmap.Map[BuildLabel, *BuildTarget]
	// Map of all currently known packages.
	packages *cmap.ErrMap[packageKey, *Package]
	// Registered subrepos, as a map of their name to their root.
	subrepos *cmap.Map[string, *Subrepo]
	// All things that subinclude anything else - either packages or build definitions that themselves subinclude other things.
	subincludes *cmap.Map[BuildLabel, []BuildLabel]
}

// AddTarget adds a new target to the graph.
func (graph *BuildGraph) AddTarget(target *BuildTarget) *BuildTarget {
	if !graph.targets.Add(target.Label, target) {
		panic("Attempted to re-add existing target to build graph: " + target.Label.String())
	}
	return target
}

// AddPackage adds a new package to the graph with given name.
func (graph *BuildGraph) AddPackage(pkg *Package) {
	key := packageKey{Name: pkg.Name, Subrepo: pkg.SubrepoName}
	if !graph.packages.Add(key, pkg) {
		panic("Attempt to re-add existing package: " + key.String())
	}
}

// Target retrieves a target from the graph by label
func (graph *BuildGraph) Target(label BuildLabel) *BuildTarget {
	return graph.targets.Get(label)
}

// TargetOrWait retrieves a target from the graph by label. If it doesn't exist yet, it returns a channel
// that can be waited upon; it will be closed after AddTarget is called for the appropriate target.
func (graph *BuildGraph) TargetOrWait(label BuildLabel) (*BuildTarget, <-chan struct{}) {
	target, ch, _ := graph.targets.GetOrWait(label)
	return target, ch
}

// TargetOrDie retrieves a target from the graph by label. Dies if the target doesn't exist.
func (graph *BuildGraph) TargetOrDie(label BuildLabel) *BuildTarget {
	target := graph.Target(label)
	if target == nil {
		log.Fatalf("Target %s not found in build graph\n", label)
	}
	return target
}

// PackageByLabel retrieves a package from the graph using the appropriate parts of the given label.
// The Name entry is ignored.
func (graph *BuildGraph) PackageByLabel(label BuildLabel) *Package {
	return graph.Package(label.PackageName, label.Subrepo)
}

// Package retrieves a package from the graph by name & subrepo, or nil if it can't be found.
func (graph *BuildGraph) Package(name, subrepo string) *Package {
	pkg, _ := graph.packages.Get(packageKey{Name: name, Subrepo: subrepo})
	return pkg
}

// PackageOrWait retrieves a package from the graph, or the error from parsing it if that failed.
// If it hasn't been parsed yet, it returns a channel that is closed once it has been, plus whether this
// caller is the first to ask for it.
func (graph *BuildGraph) PackageOrWait(label BuildLabel) (*Package, <-chan struct{}, bool, error) {
	return graph.packages.GetOrWait(packageKey{Name: label.PackageName, Subrepo: label.Subrepo})
}

// SetPackage records the result of parsing a package, releasing anything waiting on it.
func (graph *BuildGraph) SetPackage(label BuildLabel, pkg *Package, err error) {
	key := packageKey{Name: label.PackageName, Subrepo: label.Subrepo}
	if err != nil {
		graph.packages.SetError(key, err)
		return
	}
	graph.packages.Set(key, pkg)
}

// PackageOrDie retrieves a package by label, and dies if it can't be found.
func (graph *BuildGraph) PackageOrDie(label BuildLabel) *Package {
	pkg := graph.PackageByLabel(label)
	if pkg == nil {
		log.Fatalf("Package %s doesn't exist in graph", label.packageKey())
	}
	return pkg
}

// AddSubrepo adds a new subrepo to the graph. It dies if one is already registered by this name.
func (graph *BuildGraph) AddSubrepo(subrepo *Subrepo) {
	if !graph.subrepos.Add(subrepo.Name, subrepo) {
		log.Fatalf("Subrepo %s is already registered", subrepo.Name)
	}
}

// MaybeAddSubrepo adds the given subrepo to the graph, or returns the existing one if one is already registered.
func (graph *BuildGraph) MaybeAddSubrepo(subrepo *Subrepo) *Subrepo {
	if !graph.subrepos.Add(subrepo.Name, subrepo) {
		old := graph.subrepos.Get(subrepo.Name)
		if !old.Equal(subrepo) {
			log.Fatalf("Found multiple definitions for subrepo '%s' (%+v s %+v)", old.Name, old, subrepo)
		}
		return old
	}
	return subrepo
}

// Subrepo returns the subrepo with this name. It returns nil if one isn't registered.
func (graph *BuildGraph) Subrepo(name string) *Subrepo {
	return graph.subrepos.Get(name)
}

// SubrepoOrDie returns the subrepo with this name, dying if it doesn't exist.
func (graph *BuildGraph) SubrepoOrDie(name string) *Subrepo {
	subrepo := graph.Subrepo(name)
	if subrepo == nil {
		log.Fatalf("No registered subrepo by the name %s", name)
	}
	return subrepo
}

// SubrepoOrWait retrieves a subrepo from the graph.
// If it is not yet available, it returns a channel that can be waited upon.
func (graph *BuildGraph) SubrepoOrWait(subrepo string) (*Subrepo, <-chan struct{}) {
	s, wait, _ := graph.subrepos.GetOrWait(subrepo)
	return s, wait
}

// AllTargets returns a consistently ordered slice of all the targets in the graph.
func (graph *BuildGraph) AllTargets() BuildTargets {
	targets := graph.targets.Values()
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].Label.Less(targets[j].Label)
	})
	return targets
}

// AllPackages returns an iterator over all packages in the graph. They are returned in no particular order.
func (graph *BuildGraph) AllPackages() iter.Seq[*Package] {
	return func(yield func(*Package) bool) {
		for pkg, err := range graph.packages.Values() {
			if err == nil {
				if !yield(pkg) {
					break
				}
			}
		}
	}
}

// NewGraph constructs and returns a new BuildGraph.
func NewGraph() *BuildGraph {
	g := &BuildGraph{
		targets:     cmap.New[BuildLabel, *BuildTarget](cmap.DefaultShardCount, HashBuildLabel),
		packages:    cmap.NewErrMap[packageKey, *Package](cmap.DefaultShardCount, hashPackageKey, nil),
		subrepos:    cmap.New[string, *Subrepo](cmap.SmallShardCount, cmap.XXHash),
		subincludes: cmap.New[BuildLabel, []BuildLabel](cmap.DefaultShardCount, HashBuildLabel),
	}
	return g
}

// DependentTargets returns the labels that 'from' should actually depend on when it declared a dependency on 'to'.
// This is normally just 'to' but could be otherwise given require/provide shenanigans.
func (graph *BuildGraph) DependentTargets(from, to BuildLabel) []BuildLabel {
	fromTarget := graph.Target(from)
	if toTarget := graph.Target(to); fromTarget != nil && toTarget != nil {
		return toTarget.ProvideFor(fromTarget)
	}
	return []BuildLabel{to}
}

// AddSubinclude adds an edge from a label to another that it subincludes.
func (graph *BuildGraph) AddSubinclude(from, included BuildLabel) {
	graph.subincludes.Update(from, func(before []BuildLabel) []BuildLabel {
		if !slices.Contains(before, included) {
			return append(before, included)
		}
		return before
	})
}

// Subincludes returns a sequence of all targets that are subincluded from a target.
func (graph *BuildGraph) Subincludes(from BuildLabel) iter.Seq[BuildLabel] {
	return slices.Values(graph.subincludes.Get(from))
}

// SubincludeNodes returns every label that has had subincludes recorded against it.
// These are packages as well as build definitions, and include packages that are still in the process of parsing.
func (graph *BuildGraph) SubincludeNodes() iter.Seq[BuildLabel] {
	return func(yield func(BuildLabel) bool) {
		for label := range graph.subincludes.Items() {
			if !yield(label) {
				break
			}
		}
	}
}

// AllSubincludes yields every subinclude depended on from the given label, either directly or transitively.
func (graph *BuildGraph) AllSubincludes(from BuildLabel) iter.Seq[BuildLabel] {
	return func(yield func(BuildLabel) bool) {
		seen := map[BuildLabel]struct{}{from: {}}
		var f func(l BuildLabel) bool
		f = func(l BuildLabel) bool {
			for l2 := range graph.Subincludes(l) {
				if _, present := seen[l2]; present {
					continue
				}
				seen[l2] = struct{}{}
				if !yield(l2) {
					return false
				}
				if !f(l2) {
					return false
				}
			}
			return true
		}
		f(from)
	}
}
