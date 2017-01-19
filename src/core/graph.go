// Representation of the build graph.
// The graph of build targets forms a DAG which we discover from the top
// down and then build bottom-up.

package core

import (
	"fmt"
	"sort"
	"sync"
)

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
	mutex sync.RWMutex
}

// Adds a new target to the graph.
func (graph *BuildGraph) AddTarget(target *BuildTarget) *BuildTarget {
	graph.mutex.Lock()
	defer graph.mutex.Unlock()
	if _, present := graph.targets[target.Label]; present {
		panic("Attempted to re-add existing target to build graph: " + target.Label.String())
	}
	graph.targets[target.Label] = target
	// Check these reverse deps which may have already been added against this target.
	graph.linkPendingRevdeps(target.Label, target)
	if target.Label.Arch != "" {
		// Helps some stuff out to keep this guy in the graph twice.
		// TODO(pebers): are other bits of code (e.g. query) liable to be confused by this?
		noArchTarget := target.toArch("")
		graph.targets[noArchTarget.Label] = noArchTarget
		graph.linkPendingRevdeps(noArchTarget.Label, noArchTarget)
	}
	return target
}

// linkPendingRevdeps links up reverse dependencies of a label after it's added.
func (graph *BuildGraph) linkPendingRevdeps(label BuildLabel, target *BuildTarget) {
	if revdeps, present := graph.pendingRevDeps[label]; present {
		for revdep, originalTarget := range revdeps {
			if originalTarget != nil {
				graph.linkDependencies(graph.targets[revdep], originalTarget)
			} else {
				graph.linkDependencies(graph.targets[revdep], target)
			}
		}
		delete(graph.pendingRevDeps, label) // Don't need any more
	}
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
	graph.mutex.RLock()
	target := graph.targets[label]
	graph.mutex.RUnlock()
	if target == nil && label.Arch != "" {
		// Specified an architecture, we might need to clone a target at this point.
		graph.mutex.Lock()
		defer graph.mutex.Unlock()
		return graph.maybeCloneTargetForArch(label)
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
	graph.mutex.RLock()
	defer graph.mutex.RUnlock()
	return graph.packages[name]
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
	graph.mutex.RLock()
	defer graph.mutex.RUnlock()
	return len(graph.targets)
}

// Returns a sorted slice of all the targets in the graph.
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

// Used for getting a local copy of the package map without having to expose it publicly.
func (graph *BuildGraph) PackageMap() map[string]*Package {
	graph.mutex.RLock()
	defer graph.mutex.RUnlock()
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
	toTarget := graph.targets[to]
	if toTarget == nil {
		toTarget = graph.maybeCloneTargetForArch(to)
	}
	// The dependency may not exist yet if we haven't parsed its package.
	// In that case we stash it away for later.
	if toTarget == nil {
		graph.addPendingRevDep(from, to, nil)
	} else {
		graph.linkDependencies(fromTarget, toTarget)
	}
}

func NewGraph() *BuildGraph {
	return &BuildGraph{
		targets:        make(map[BuildLabel]*BuildTarget),
		packages:       make(map[string]*Package),
		pendingRevDeps: make(map[BuildLabel]map[BuildLabel]*BuildTarget),
		revDeps:        make(map[BuildLabel][]*BuildTarget),
	}
}

// ReverseDependencies returns the set of revdeps on the given target.
func (graph *BuildGraph) ReverseDependencies(target *BuildTarget) []*BuildTarget {
	graph.mutex.RLock()
	defer graph.mutex.RUnlock()
	if revdeps, present := graph.revDeps[target.Label]; present {
		return revdeps[:]
	}
	return nil
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
// Also at this point we resolve any architecture differences if cross-compiling is needed.
func (graph *BuildGraph) linkDependencies(fromTarget, toTarget *BuildTarget) {
	if fromTarget.Label.Arch != "" && toTarget.Label.Arch != "" && fromTarget.Label.Arch != toTarget.Label.Arch {
		panic(fmt.Sprintf("%s requires a dependency on %s, but it's not available for that architecture", fromTarget.Label, toTarget.Label.toArch("")))
	}
	if fromTarget.Label.Arch != "" && !fromTarget.IsTool(toTarget.Label) {
		// Source dependency from a target that's not being built for the host.
		toTarget = graph.cloneTargetForArch(toTarget, fromTarget.Label.Arch)
	}
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

// cloneTargetForArch returns a build target for the given architecture. It's otherwise
// identical to the original one.
func (graph *BuildGraph) cloneTargetForArch(target *BuildTarget, arch string) *BuildTarget {
	t, present := graph.targets[target.Label.toArch(arch)]
	if present {
		return t
	}
	t = target.toArch(arch)
	graph.targets[t.Label] = t
	return t
}

// maybeCloneTargetForArch returns a build target for the given architecture, if a no-arch version
// exists in the graph already.
func (graph *BuildGraph) maybeCloneTargetForArch(label BuildLabel) *BuildTarget {
	if label.Arch == "" {
		return nil
	}
	target := graph.targets[label.noArch()]
	if target == nil {
		return nil
	}
	return graph.cloneTargetForArch(target, label.Arch)
}

func (graph *BuildGraph) addPendingRevDep(from, to BuildLabel, orig *BuildTarget) {
	to.Arch = "" // Pending revdeps never have an associated architecture.
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
