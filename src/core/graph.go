// Representation of the build graph.
// The graph of build targets forms a DAG which we discover from the top
// down and then build bottom-up.

package core

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sort"
	"sync"

	"github.com/thought-machine/please/src/cmap"
)

type labelSet map[BuildLabel]struct{}

func (ls labelSet) add(l BuildLabel) {
	ls[l] = struct{}{}
}

func (ls labelSet) contains(l BuildLabel) bool {
	_, ok := ls[l]
	return ok
}

// A BuildGraph contains all the loaded targets and packages and maintains their
// relationships, especially reverse dependencies which are calculated here.
type BuildGraph struct {
	// Map of all currently known targets by their label.
	targets *cmap.Map[BuildLabel, *BuildTarget]
	// Map of all currently known packages.
	packages *cmap.ErrMap[packageKey, *Package]
	// Registered subrepos, as a map of their name to their root.
	subrepos *cmap.Map[string, *Subrepo]
	// Subincludes that are subincluded by other subincludes
	subincludeSubincludes map[BuildLabel]labelSet
	// Use a mutex as a labelSet isn't atomic. We need to guard against inserting as well as mutating the value.
	subincMux sync.Mutex
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

// GetOrSetPackage retrieves a package from the graph.
// If it doesn't exist, it calls the supplied function to create it.
func (graph *BuildGraph) GetOrSetPackage(ctx context.Context, label BuildLabel, f func() (*Package, error)) (*Package, error) {
	return graph.packages.GetOrSetCtx(ctx, packageKey{Name: label.PackageName, Subrepo: label.Subrepo}, f)
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

// PackageMap returns a map of name to package.
// TODO(peterebden): Change this to an iterator.
func (graph *BuildGraph) PackageMap() map[string]*Package {
	packages := map[string]*Package{}
	graph.packages.Range(func(k packageKey, v *Package) {
		packages[k.String()] = v
	})
	return packages
}

// NewGraph constructs and returns a new BuildGraph.
func NewGraph() *BuildGraph {
	g := &BuildGraph{
		targets:               cmap.New[BuildLabel, *BuildTarget](cmap.DefaultShardCount, hashBuildLabel),
		packages:              cmap.NewErrMap[packageKey, *Package](cmap.DefaultShardCount, hashPackageKey, nil),
		subrepos:              cmap.New[string, *Subrepo](cmap.SmallShardCount, cmap.XXHash),
		subincludeSubincludes: map[BuildLabel]labelSet{},
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

// TransitiveSubincludes returns all the subincludes made by a given subinclude
func (graph *BuildGraph) TransitiveSubincludes(l BuildLabel) []BuildLabel {
	graph.subincMux.Lock()
	defer graph.subincMux.Unlock()

	incs := labelSet{}
	graph.findTransitiveSubincludes(l, incs)

	ls := slices.Collect(maps.Keys(incs))
	sort.Sort(BuildLabels(ls))
	return ls
}

func (graph *BuildGraph) findTransitiveSubincludes(label BuildLabel, includes labelSet) {
	if includes.contains(label) {
		return
	}
	includes.add(label)
	for l := range graph.subincludeSubincludes[label] {
		graph.findTransitiveSubincludes(l, includes)
	}
}

func (graph *BuildGraph) RegisterTransitiveSubinclude(from, to BuildLabel) {
	graph.subincMux.Lock()
	defer graph.subincMux.Unlock()

	incs, ok := graph.subincludeSubincludes[from]
	if !ok {
		incs = labelSet{}
		graph.subincludeSubincludes[from] = incs
	}
	incs.add(to)
}
