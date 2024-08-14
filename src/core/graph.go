// Representation of the build graph.
// The graph of build targets forms a DAG which we discover from the top
// down and then build bottom-up.

package core

import (
	"sort"

	"github.com/thought-machine/please/src/cmap"
)

// A BuildGraph contains all the loaded targets and packages and maintains their
// relationships, especially reverse dependencies which are calculated here.
type BuildGraph struct {
	// Map of all currently known targets by their label.
	targets *cmap.Map[BuildLabel, *BuildTarget]
	// Map of all currently known packages.
	packages *cmap.Map[packageKey, *Package]
	// Registered subrepos, as a map of their name to their root.
	subrepos *cmap.Map[string, *Subrepo]
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

// TargetOrDie retrieves a target from the graph by label. Dies if the target doesn't exist.
func (graph *BuildGraph) TargetOrDie(label BuildLabel) *BuildTarget {
	target := graph.Target(label)
	if target == nil {
		log.Fatalf("Target %s not found in build graph\n", label)
	}
	return target
}

// WaitForTarget returns the given target, waiting for it to be added if it isn't yet.
// It returns nil if the target finally turns out not to exist.
func (graph *BuildGraph) WaitForTarget(label BuildLabel) *BuildTarget {
	t, tch, _ := graph.targets.GetOrWait(label)
	if t != nil {
		return t
	}
	p, pch, _ := graph.packages.GetOrWait(packageKey{Name: label.PackageName, Subrepo: label.Subrepo})
	if p != nil {
		// Check target again to avoid race conditions
		return graph.Target(label)
	}
	// Now we need to wait for either (hopefully) the target or its package to exist.
	// Either the target will, which is fine, or if the package appears but the target doesn't
	// we will conclude it doesn't exist.
	select {
	case <-tch:
	case <-pch:
	}
	return graph.Target(label)
}

// PackageByLabel retrieves a package from the graph using the appropriate parts of the given label.
// The Name entry is ignored.
func (graph *BuildGraph) PackageByLabel(label BuildLabel) *Package {
	return graph.Package(label.PackageName, label.Subrepo)
}

// Package retrieves a package from the graph by name & subrepo, or nil if it can't be found.
func (graph *BuildGraph) Package(name, subrepo string) *Package {
	return graph.packages.Get(packageKey{Name: name, Subrepo: subrepo})
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

// AllTargets returns a consistently ordered slice of all the targets in the graph.
func (graph *BuildGraph) AllTargets() BuildTargets {
	targets := graph.targets.Values()
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].Label.Less(targets[j].Label)
	})
	return targets
}

// PackageMap returns a copy of the graph's internal map of name to package.
func (graph *BuildGraph) PackageMap() map[string]*Package {
	packages := map[string]*Package{}
	for _, pkg := range graph.packages.Values() {
		packages[packageKey{Subrepo: pkg.SubrepoName, Name: pkg.Name}.String()] = pkg
	}
	return packages
}

// NewGraph constructs and returns a new BuildGraph.
func NewGraph() *BuildGraph {
	g := &BuildGraph{
		targets:  cmap.New[BuildLabel, *BuildTarget](cmap.DefaultShardCount, hashBuildLabel),
		packages: cmap.New[packageKey, *Package](cmap.DefaultShardCount, hashPackageKey),
		subrepos: cmap.New[string, *Subrepo](cmap.SmallShardCount, cmap.XXHash),
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
