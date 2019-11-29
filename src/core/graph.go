// Representation of the build graph.
// The graph of build targets forms a DAG which we discover from the top
// down and then build bottom-up.

package core

import (
	"reflect"
	"sort"
	"sync"
)

// A BuildGraph contains all the loaded targets and packages and maintains their
// relationships, especially reverse dependencies which are calculated here.
// It also arbitrates access to a lot of things via its builtin mutex which
// is probably our most overused lock :(
type BuildGraph struct {
	// Map of all currently known targets by their label.
	targets map[BuildLabel]*BuildTarget
	// Map of all currently known packages.
	packages map[packageKey]*Package
	// Reverse dependencies that are pending on targets actually being added to the graph.
	pendingRevDeps map[BuildLabel]map[BuildLabel]*BuildTarget
	// Actual reverse dependencies
	revDeps map[BuildLabel][]*BuildTarget
	// Registered subrepos, as a map of their name to their root.
	subrepos map[string]*Subrepo
	// Used to arbitrate access to the graph. We parallelise most build operations
	// and Go maps aren't natively threadsafe so this is needed.
	mutex sync.RWMutex
}

// AddTarget adds a new target to the graph.
func (graph *BuildGraph) AddTarget(target *BuildTarget) *BuildTarget {
	graph.mutex.Lock()
	defer graph.mutex.Unlock()
	if _, present := graph.targets[target.Label]; present {
		panic("Attempted to re-add existing target to build graph: " + target.Label.String())
	}
	graph.targets[target.Label] = target
	// Register any of its dependencies now
	for _, dep := range target.DeclaredDependencies() {
		graph.addDependencyForTarget(target, dep)
	}
	// Check these reverse deps which may have already been added against this target.
	revdeps, present := graph.pendingRevDeps[target.Label]
	if present {
		for revdep, originalTarget := range revdeps {
			if originalTarget != nil {
				graph.linkDependencies(graph.targets[revdep], originalTarget)
			} else {
				graph.linkDependencies(graph.targets[revdep], target)
			}
		}
		delete(graph.pendingRevDeps, target.Label) // Don't need any more
	}
	return target
}

// AddPackage adds a new package to the graph with given name.
func (graph *BuildGraph) AddPackage(pkg *Package) {
	key := packageKey{Name: pkg.Name, Subrepo: pkg.SubrepoName}
	graph.mutex.Lock()
	defer graph.mutex.Unlock()
	if _, present := graph.packages[key]; present {
		panic("Attempt to readd existing package: " + key.String())
	}
	graph.packages[key] = pkg
}

// Target retrieves a target from the graph by label
func (graph *BuildGraph) Target(label BuildLabel) *BuildTarget {
	graph.mutex.RLock()
	defer graph.mutex.RUnlock()
	return graph.targets[label]
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
	graph.mutex.RLock()
	defer graph.mutex.RUnlock()
	return graph.packages[packageKey{Name: name, Subrepo: subrepo}]
}

// PackageOrDie retrieves a package by label, and dies if it can't be found.
func (graph *BuildGraph) PackageOrDie(label BuildLabel) *Package {
	pkg := graph.PackageByLabel(label)
	if pkg == nil {
		log.Fatalf("Package %s doesn't exist in graph", packageKey{Name: label.PackageName, Subrepo: label.Subrepo})
	}
	return pkg
}

// AddSubrepo adds a new subrepo to the graph. It dies if one is already registered by this name.
func (graph *BuildGraph) AddSubrepo(subrepo *Subrepo) {
	graph.mutex.Lock()
	defer graph.mutex.Unlock()
	if _, present := graph.subrepos[subrepo.Name]; present {
		log.Fatalf("Subrepo %s is already registered", subrepo.Name)
	}
	graph.subrepos[subrepo.Name] = subrepo
}

// MaybeAddSubrepo adds the given subrepo to the graph, or returns the existing one if one is already registered.
func (graph *BuildGraph) MaybeAddSubrepo(subrepo *Subrepo) *Subrepo {
	graph.mutex.Lock()
	defer graph.mutex.Unlock()
	if s, present := graph.subrepos[subrepo.Name]; present {
		if !reflect.DeepEqual(s, subrepo) {
			log.Fatalf("Found multiple definitions for subrepo '%s' (%+v s %+v)",
				s.Name, s, subrepo)
		}
		return s
	}
	graph.subrepos[subrepo.Name] = subrepo
	return subrepo
}

// Subrepo returns the subrepo with this name. It returns nil if one isn't registered.
func (graph *BuildGraph) Subrepo(name string) *Subrepo {
	graph.mutex.RLock()
	defer graph.mutex.RUnlock()
	return graph.subrepos[name]
}

// SubrepoOrDie returns the subrepo with this name, dying if it doesn't exist.
func (graph *BuildGraph) SubrepoOrDie(name string) *Subrepo {
	subrepo := graph.Subrepo(name)
	if subrepo == nil {
		log.Fatalf("No registered subrepo by the name %s", name)
	}
	return subrepo
}

// Len returns the number of targets currently in the graph.
func (graph *BuildGraph) Len() int {
	graph.mutex.RLock()
	defer graph.mutex.RUnlock()
	return len(graph.targets)
}

// AllTargets returns a consistently ordered slice of all the targets in the graph.
func (graph *BuildGraph) AllTargets() BuildTargets {
	graph.mutex.RLock()
	defer graph.mutex.RUnlock()
	targets := make(BuildTargets, 0, len(graph.targets))
	for _, target := range graph.targets {
		targets = append(targets, target)
	}
	sort.Sort(targets)
	return targets
}

// PackageMap returns a copy of the graph's internal map of name to package.
func (graph *BuildGraph) PackageMap() map[string]*Package {
	graph.mutex.RLock()
	defer graph.mutex.RUnlock()
	packages := make(map[string]*Package, len(graph.packages))
	for k, v := range graph.packages {
		packages[k.String()] = v
	}
	return packages
}

// AddDependency adds a dependency between two build targets.
// The 'to' target doesn't necessarily have to exist in the graph yet (but 'from' must).
func (graph *BuildGraph) AddDependency(from BuildLabel, to BuildLabel) {
	graph.mutex.Lock()
	defer graph.mutex.Unlock()
	graph.addDependencyForTarget(graph.targets[from], to)
}

// addDependencyForTarget adds a dependency between two build targets.
// The 'to' target doesn't necessarily have to exist in the graph yet.
// The caller must already hold the lock before calling this.
func (graph *BuildGraph) addDependencyForTarget(fromTarget *BuildTarget, to BuildLabel) {
	// We might have done this already; do a quick check here first.
	if fromTarget.hasResolvedDependency(to) {
		return
	}
	toTarget, present := graph.targets[to]
	// The dependency may not exist yet if we haven't parsed its package.
	// In that case we stash it away for later.
	if !present {
		graph.addPendingRevDep(fromTarget.Label, to, nil)
	} else {
		graph.linkDependencies(fromTarget, toTarget)
	}
}

// NewGraph constructs and returns a new BuildGraph.
// Users should not attempt to construct one themselves.
func NewGraph() *BuildGraph {
	return &BuildGraph{
		targets:        map[BuildLabel]*BuildTarget{},
		packages:       map[packageKey]*Package{},
		pendingRevDeps: map[BuildLabel]map[BuildLabel]*BuildTarget{},
		revDeps:        map[BuildLabel][]*BuildTarget{},
		subrepos:       map[string]*Subrepo{},
	}
}

// ReverseDependencies returns the set of revdeps on the given target.
func (graph *BuildGraph) ReverseDependencies(target *BuildTarget) []*BuildTarget {
	graph.mutex.RLock()
	defer graph.mutex.RUnlock()
	if revdeps, present := graph.revDeps[target.Label]; present {
		return revdeps[:]
	}
	return []*BuildTarget{}
}

// AllDepsBuilt returns true if all the dependencies of a target are built.
func (graph *BuildGraph) AllDepsBuilt(target *BuildTarget) bool {
	graph.mutex.RLock()
	defer graph.mutex.RUnlock()
	return target.allDepsBuilt()
}

// AllDependenciesResolved returns true once all the dependencies of a target have been
// parsed and resolved to real targets.
func (graph *BuildGraph) AllDependenciesResolved(target *BuildTarget) bool {
	graph.mutex.RLock()
	defer graph.mutex.RUnlock()
	return target.allDependenciesResolved()
}

// linkDependencies adds the dependency of fromTarget on toTarget and the corresponding
// reverse dependency in the other direction.
// This is complicated somewhat by the require/provide mechanism which is resolved at this
// point, but some of the dependencies may not yet exist.
func (graph *BuildGraph) linkDependencies(fromTarget, toTarget *BuildTarget) {
	for _, label := range toTarget.ProvideFor(fromTarget) {
		if target, present := graph.targets[label]; present {
			fromTarget.resolveDependency(toTarget.Label, target)
			graph.revDeps[label] = append(graph.revDeps[label], fromTarget)
		} else {
			graph.addPendingRevDep(fromTarget.Label, label, toTarget)
		}
	}
}

func (graph *BuildGraph) addPendingRevDep(from, to BuildLabel, orig *BuildTarget) {
	if deps, present := graph.pendingRevDeps[to]; present {
		deps[from] = orig
	} else {
		graph.pendingRevDeps[to] = map[BuildLabel]*BuildTarget{from: orig}
	}
}

// DependentTargets returns the labels that 'from' should actually depend on when it declared a dependency on 'to'.
// This is normally just 'to' but could be otherwise given require/provide shenanigans.
func (graph *BuildGraph) DependentTargets(from, to BuildLabel) []BuildLabel {
	fromTarget := graph.Target(from)
	if toTarget := graph.Target(to); fromTarget != nil && toTarget != nil {
		graph.mutex.Lock()
		defer graph.mutex.Unlock()
		return toTarget.ProvideFor(fromTarget)
	}
	return []BuildLabel{to}
}
