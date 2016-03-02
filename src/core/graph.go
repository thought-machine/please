// Representation of the build graph.
// The graph of build targets forms a DAG which we discover from the top
// down and then build bottom-up.

package core

import "sort"
import "sync"

type BuildGraph struct {
	// Map of all currently known targets by their label.
	targets map[BuildLabel]*BuildTarget
	// Map of all currently known packages.
	packages map[string]*Package
	// Reverse dependencies that are pending on targets actually being added to the graph.
	pendingRevDeps map[BuildLabel]map[BuildLabel]*BuildTarget
	// Actual reverse dependencies
	revDeps map[BuildLabel][]*BuildTarget
	// Used to arbitrate access to the graph. We parallelise most build operations
	// and Go maps aren't natively threadsafe so this is needed.
	mutex sync.Mutex
}

// Adds a new target to the graph.
func (graph *BuildGraph) AddTarget(target *BuildTarget) *BuildTarget {
	graph.mutex.Lock()
	defer graph.mutex.Unlock()
	_, present := graph.targets[target.Label]
	if present {
		panic("Attempted to re-add existing target to build graph: " + target.Label.String())
	}
	graph.targets[target.Label] = target
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

// Adds a new package to the graph with given name.
func (graph *BuildGraph) AddPackage(pkg *Package) {
	graph.mutex.Lock()
	defer graph.mutex.Unlock()
	if _, present := graph.packages[pkg.Name]; present {
		panic("Attempt to readd existing package: " + pkg.Name)
	}
	graph.packages[pkg.Name] = pkg
}

// Target retrieves a target from the graph by label
func (graph *BuildGraph) Target(label BuildLabel) *BuildTarget {
	graph.mutex.Lock()
	defer graph.mutex.Unlock()
	target, present := graph.targets[label]
	if !present {
		target = nil // easier than a 'null' BuildTarget.
	}
	return target
}

// TargetOrDie retrieves a target from the graph by label. Dies if the target doesn't exist.
func (graph *BuildGraph) TargetOrDie(label BuildLabel) *BuildTarget {
	target := graph.Target(label)
	if target == nil {
		log.Fatalf("Target %s not found in build graph\n", label)
	}
	return target
}

// Package retrieves a package from the graph by name
func (graph *BuildGraph) Package(name string) *Package {
	graph.mutex.Lock()
	defer graph.mutex.Unlock()
	pkg, present := graph.packages[name]
	if !present {
		pkg = nil
	}
	return pkg
}

// PackageOrDie retrieves a package by name, and dies if it can't be found.
func (graph *BuildGraph) PackageOrDie(name string) *Package {
	pkg := graph.Package(name)
	if pkg == nil {
		log.Fatalf("Package %s doesn't exist in graph", name)
	}
	return pkg
}

func (graph *BuildGraph) Len() int {
	graph.mutex.Lock()
	defer graph.mutex.Unlock()
	return len(graph.targets)
}

// Returns a sorted slice of all the targets in the graph.
func (graph *BuildGraph) AllTargets() BuildTargets {
	graph.mutex.Lock()
	defer graph.mutex.Unlock()
	targets := make(BuildTargets, 0, len(graph.targets))
	for _, target := range graph.targets {
		targets = append(targets, target)
	}
	sort.Sort(targets)
	return targets
}

// Used for getting a local copy of the package map without having to expose it publicly.
func (graph *BuildGraph) PackageMap() map[string]*Package {
	graph.mutex.Lock()
	defer graph.mutex.Unlock()
	packages := make(map[string]*Package)
	for name, pkg := range graph.packages {
		packages[name] = pkg
	}
	return packages
}

func (graph *BuildGraph) AddDependency(from BuildLabel, to BuildLabel) {
	graph.mutex.Lock()
	defer graph.mutex.Unlock()
	fromTarget := graph.targets[from]
	// We might have done this already; do a quick check here first.
	if fromTarget.hasResolvedDependency(to) {
		return
	}
	toTarget, present := graph.targets[to]
	// The dependency may not exist yet if we haven't parsed its package.
	// In that case we stash it away for later.
	if !present {
		graph.addPendingRevDep(from, to, nil)
	} else {
		graph.linkDependencies(fromTarget, toTarget)
	}
}

func NewGraph() *BuildGraph {
	graph := new(BuildGraph)
	graph.targets = make(map[BuildLabel]*BuildTarget)
	graph.packages = make(map[string]*Package)
	graph.pendingRevDeps = make(map[BuildLabel]map[BuildLabel]*BuildTarget)
	graph.revDeps = make(map[BuildLabel][]*BuildTarget)
	return graph
}

// ReverseDependencies returns the set of revdeps on the given target.
func (graph *BuildGraph) ReverseDependencies(target *BuildTarget) []*BuildTarget {
	graph.mutex.Lock()
	defer graph.mutex.Unlock()
	if revdeps, present := graph.revDeps[target.Label]; present {
		return revdeps[:]
	}
	return []*BuildTarget{}
}

// AllDepsBuilt returns true if all the dependencies of a target are built.
func (graph *BuildGraph) AllDepsBuilt(target *BuildTarget) bool {
	graph.mutex.Lock()
	defer graph.mutex.Unlock()
	return target.allDepsBuilt()
}

// AllDependenciesResolved returns true once all the dependencies of a target have been
// parsed and resolved to real targets.
func (graph *BuildGraph) AllDependenciesResolved(target *BuildTarget) bool {
	graph.mutex.Lock()
	defer graph.mutex.Unlock()
	return target.allDependenciesResolved()
}

// linkDependencies adds the dependency of fromTarget on toTarget and the corresponding
// reverse dependency in the other direction.
// This is complicated somewhat by the require/provide mechanism which is resolved at this
// point, but some of the dependencies may not yet exist.
func (graph *BuildGraph) linkDependencies(fromTarget, toTarget *BuildTarget) {
	for _, label := range toTarget.ProvideFor(fromTarget) {
		target, present := graph.targets[label]
		if present {
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
