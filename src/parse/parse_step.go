// Package responsible for parsing build files and constructing build targets & the graph.
//
// The actual work here is done by an embedded PyPy instance. Various rules are built in to
// the binary itself using go-bindata to embed the .py files; these are always available to
// all programs which is rather nice, but it does mean that must be run before 'go run' etc
// will work as expected.
package parse

import (
	"fmt"
	"os"
	"path"
	"strings"
	"sync"

	"core"
)

// Parses the package corresponding to a single build label. The label can be :all to add all targets in a package.
// It is not an error if the package has already been parsed.
//
// By default, after the package is parsed, any targets that are now needed for the build and ready
// to be built are queued, and any new packages are queued for parsing. When a specific label is requested
// this is straightforward, but when parsing for pseudo-targets like :all and ..., various flags affect it:
// If 'noDeps' is true, then no new packages will be added and no new targets queued.
// 'include' and 'exclude' refer to the labels of targets to be added. If 'include' is non-empty then only
// targets with at least one matching label are added. Any targets with a label in 'exclude' are not added.
func Parse(tid int, state *core.BuildState, label, dependor core.BuildLabel, noDeps bool, include, exclude []string) {
	defer func() {
		if r := recover(); r != nil {
			state.LogBuildError(tid, label, core.ParseFailed, fmt.Errorf("%s", r), "Failed to parse package")
		}
	}()
	// First see if this package already exists; once it's in the graph it will have been parsed.
	pkg := state.Graph.Package(label.PackageName)
	if pkg != nil {
		// Does exist, all we need to do is toggle on this target
		activateTarget(state, pkg, label, dependor, noDeps, include, exclude)
		return
	}
	// We use the name here to signal undeferring of a package. If we get that we need to retry the package regardless.
	if dependor.Name != "_UNDEFER_" && !firstToParse(label, dependor) {
		// Check this again to avoid a potential race
		if pkg = state.Graph.Package(label.PackageName); pkg != nil {
			activateTarget(state, pkg, label, dependor, noDeps, include, exclude)
		} else {
			log.Debug("Adding pending parse for %s", label)
		}
		return
	}
	// If we get here then it falls to us to parse this package
	state.LogBuildResult(tid, label, core.PackageParsing, "Parsing...")
	pkg = parsePackage(state, label, dependor)
	if pkg == nil {
		state.LogBuildResult(tid, label, core.PackageParsed, "Deferred")
		return
	}

	// Now add any lurking pending targets for this package.
	pendingTargetMutex.Lock()
	pending := pendingTargets[label.PackageName]                       // Must be present.
	pendingTargets[label.PackageName] = map[string][]core.BuildLabel{} // Empty this to free memory, but leave a sentinel
	pendingTargetMutex.Unlock()                                        // Nothing will look up this package in the map again.
	for targetName, dependors := range pending {
		for _, dependor := range dependors {
			lbl := core.BuildLabel{PackageName: label.PackageName, Name: targetName}
			activateTarget(state, pkg, lbl, dependor, noDeps, include, exclude)
		}
	}
	state.LogBuildResult(tid, label, core.PackageParsed, "Parsed")
}

// activateTarget marks a target as active (ie. to be built) and adds its dependencies as pending parses.
func activateTarget(state *core.BuildState, pkg *core.Package, label, dependor core.BuildLabel, noDeps bool, include, exclude []string) {
	if !label.IsAllTargets() && state.Graph.Target(label) == nil {
		msg := fmt.Sprintf("Parsed build file %s/BUILD but it doesn't contain target %s", label.PackageName, label.Name)
		if dependor != core.OriginalTarget {
			msg += fmt.Sprintf(" (depended on by %s)", dependor)
		}
		panic(msg + suggestTargets(pkg, label, dependor))
	}
	if noDeps && !dependor.IsAllTargets() { // IsAllTargets indicates requirement for parse
		return // Some kinds of query don't need a full recursive parse.
	} else if label.IsAllTargets() {
		for _, target := range pkg.Targets {
			if target.ShouldInclude(include, exclude) {
				// Must always do this for coverage because we need to calculate sources of
				// non-test targets later on.
				if !state.NeedTests || target.IsTest || state.NeedCoverage {
					addDep(state, target.Label, dependor, false, dependor.IsAllTargets())
				}
			}
		}
	} else {
		for _, l := range state.Graph.DependentTargets(dependor, label) {
			// We use :all to indicate a dependency needed for parse.
			addDep(state, l, dependor, false, dependor.IsAllTargets())
		}
	}
}

// Used to arbitrate single access to these maps
var pendingTargetMutex sync.Mutex

// Map of package name -> target name -> label that requested parse
var pendingTargets = map[string]map[string][]core.BuildLabel{}

// Map of package name -> target name -> package names that're waiting for it
var deferredParses = map[string]map[string][]string{}

// firstToParse returns true if the caller is the first to parse a given package and hence should
// continue parsing that file. It only returns true once for each package but stores subsequent
// targets in the pendingTargets map.
func firstToParse(label, dependor core.BuildLabel) bool {
	pendingTargetMutex.Lock()
	defer pendingTargetMutex.Unlock()
	if pkg, present := pendingTargets[label.PackageName]; present {
		pkg[label.Name] = append(pkg[label.Name], dependor)
		return false
	}
	pendingTargets[label.PackageName] = map[string][]core.BuildLabel{label.Name: []core.BuildLabel{dependor}}
	return true
}

// deferParse defers the parsing of a package until the given label has been built.
// Returns true if it was deferred, or false if it's already built.
func deferParse(label core.BuildLabel, pkg *core.Package) bool {
	pendingTargetMutex.Lock()
	defer pendingTargetMutex.Unlock()
	if target := core.State.Graph.Target(label); target != nil && target.State() >= core.Built {
		return false
	}
	log.Debug("Deferring parse of %s pending %s", pkg.Name, label)
	if m, present := deferredParses[label.PackageName]; present {
		m[label.Name] = append(m[label.Name], pkg.Name)
	} else {
		deferredParses[label.PackageName] = map[string][]string{label.Name: []string{pkg.Name}}
	}
	core.State.AddPendingParse(label, core.BuildLabel{PackageName: pkg.Name, Name: "all"}, true)
	return true
}

// UndeferAnyParses un-defers the parsing of a package if it depended on some subinclude target being built.
func UndeferAnyParses(state *core.BuildState, target *core.BuildTarget) {
	pendingTargetMutex.Lock()
	defer pendingTargetMutex.Unlock()
	if m, present := deferredParses[target.Label.PackageName]; present {
		if s, present := m[target.Label.Name]; present {
			for _, deferredPackageName := range s {
				log.Debug("Undeferring parse of %s", deferredPackageName)
				state.AddPendingParse(
					core.BuildLabel{PackageName: deferredPackageName, Name: getDependingTarget(deferredPackageName)},
					core.BuildLabel{PackageName: deferredPackageName, Name: "_UNDEFER_"},
					false,
				)
			}
			delete(m, target.Label.Name) // Don't need this any more
		}
	}
}

// getDependingTarget returns the name of any one target in packageName that required parsing.
func getDependingTarget(packageName string) string {
	// We need to supply a label in this package that actually needs to be built.
	// Fortunately there must be at least one of these in the pending target map...
	if m, present := pendingTargets[packageName]; present {
		for target := range m {
			return target
		}
	}
	// We shouldn't really get here, of course.
	log.Errorf("No pending target entry for %s at deferral. Must assume :all.", packageName)
	return "all"
}

// parsePackage performs the initial parse of a package.
// It's assumed that the caller used firstToParse to ascertain that they only call this once per package.
func parsePackage(state *core.BuildState, label, dependor core.BuildLabel) *core.Package {
	packageName := label.PackageName
	pkg := core.NewPackage(packageName)
	if pkg.Filename = buildFileName(state, packageName); pkg.Filename == "" {
		exists := core.PathExists(packageName)
		// Handle quite a few cases to provide more obvious error messages.
		if dependor != core.OriginalTarget && exists {
			panic(fmt.Sprintf("%s depends on %s, but there's no BUILD file in %s/", dependor, label, packageName))
		} else if dependor != core.OriginalTarget {
			panic(fmt.Sprintf("%s depends on %s, but the directory %s doesn't exist", dependor, label, packageName))
		} else if exists {
			panic(fmt.Sprintf("Can't build %s; there's no BUILD file in %s/", label, packageName))
		}
		panic(fmt.Sprintf("Can't build %s; the directory %s doesn't exist", label, packageName))
	}

	if parsePackageFile(state, pkg.Filename, pkg) {
		return nil // Indicates deferral
	}

	for _, target := range pkg.Targets {
		state.Graph.AddTarget(target)
		for _, out := range target.DeclaredOutputs() {
			pkg.MustRegisterOutput(out, target)
		}
		for _, out := range target.TestOutputs {
			pkg.MustRegisterOutput(out, target)
		}
	}
	// Do this in a separate loop so we get intra-package dependencies right now.
	for _, target := range pkg.Targets {
		for _, dep := range target.DeclaredDependencies() {
			state.Graph.AddDependency(target.Label, dep)
		}
	}
	state.Graph.AddPackage(pkg) // Calling this means nobody else will add entries to pendingTargets for this package.
	createInitPyIfNeeded(pkg, packageName)
	return pkg
}

// Pre-emptively create __init__.py files so we can load generated modules dynamically.
// It's a bit cheeky to do language-specific logic here but it's hard to find another way.
func createInitPyIfNeeded(pkg *core.Package, packagePath string) {
	for _, target := range pkg.Targets {
		for _, require := range target.Requires {
			if require == "py" {
				dir := path.Join(core.RepoRoot, "plz-out/gen", packagePath)
				for i := 0; i < len(strings.Split(packagePath, "/")); i++ {
					initPy := path.Join(dir, "__init__.py")
					if !core.PathExists(initPy) {
						if err := os.MkdirAll(dir, core.DirPermissions); err != nil {
							panic(fmt.Sprintf("Failed to create directory %s: %s", dir, err))
						} else if file, err := os.Create(initPy); err != nil {
							panic(fmt.Sprintf("Failed to create file %s: %s", initPy, err))
						} else {
							file.Close()
						}
					}
					dir = path.Dir(dir)
				}
				return
			}
		}
	}
}

func buildFileName(state *core.BuildState, pkgName string) string {
	// Bazel defines targets in its "external" package from its WORKSPACE file.
	// We will fake this by treating that as an actual package file...
	if state.Config.Bazel.Compatibility && pkgName == "external" {
		return "WORKSPACE"
	}
	for _, buildFileName := range state.Config.Please.BuildFileName {
		if filename := path.Join(pkgName, buildFileName); core.FileExists(filename) {
			return filename
		}
	}
	return ""
}

// Adds a single target to the build queue.
func addDep(state *core.BuildState, label, dependor core.BuildLabel, rescan, forceBuild bool) {
	// Stop at any package that's not loaded yet
	if state.Graph.Package(label.PackageName) == nil {
		state.AddPendingParse(label, dependor, false)
		return
	}
	target := state.Graph.Target(label)
	if target == nil {
		log.Fatalf("Target %s (referenced by %s) doesn't exist\n", label, dependor)
	}
	if target.State() >= core.Active && !rescan && !forceBuild {
		return // Target is already tagged to be built and likely on the queue.
	}
	// Only do this bit if we actually need to build the target
	if !target.SyncUpdateState(core.Inactive, core.Semiactive) && !rescan && !forceBuild {
		return
	}
	log.Debug("Activating target %s (depended on by %s)", label, dependor)
	if state.NeedBuild || forceBuild {
		if target.SyncUpdateState(core.Semiactive, core.Active) {
			state.AddActiveTarget()
			if target.IsTest && state.NeedTests {
				state.AddActiveTarget() // Tests count twice if we're gonna run them.
			}
		}
	}
	// If this target has no deps, add it to the queue now, otherwise handle its deps.
	// Only add if we need to build targets (not if we're just parsing) but we might need it to parse...
	if target.State() == core.Active && state.Graph.AllDepsBuilt(target) {
		if target.SyncUpdateState(core.Active, core.Pending) {
			state.AddPendingBuild(label, dependor.IsAllTargets())
		}
		if !rescan {
			return
		}
	}
	for _, dep := range target.DeclaredDependencies() {
		// Check the require/provide stuff; we may need to add a different target.
		if len(target.Requires) > 0 {
			if depTarget := state.Graph.Target(dep); depTarget != nil && len(depTarget.Provides) > 0 {
				for _, provided := range depTarget.ProvideFor(target) {
					addDep(state, provided, label, false, forceBuild)
				}
				continue
			}
		}
		addDep(state, dep, label, false, forceBuild)
	}
}

// RunPreBuildFunction runs a pre-build callback function registered on a build target via pre_build = <...>.
//
// This is called before the target is built. It doesn't receive any output like the post-build one does but can
// be useful for other things; for example if you want to investigate a target's transitive labels to adjust
// its build command, you have to do that here (because in general the transitive dependencies aren't known
// when the rule is evaluated).
func RunPreBuildFunction(tid int, state *core.BuildState, target *core.BuildTarget) error {
	state.LogBuildResult(tid, target.Label, core.PackageParsing,
		fmt.Sprintf("Running pre-build function for %s", target.Label))
	pkg := state.Graph.Package(target.Label.PackageName)
	pkg.BuildCallbackMutex.Lock()
	defer pkg.BuildCallbackMutex.Unlock()
	if err := runPreBuildFunction(pkg, target); err != nil {
		state.LogBuildError(tid, target.Label, core.ParseFailed, err, "Failed pre-build function for %s", target.Label)
		return err
	}
	rescanDeps(state, pkg)
	state.LogBuildResult(tid, target.Label, core.TargetBuilding,
		fmt.Sprintf("Finished pre-build function for %s", target.Label))
	return nil
}

// RunPostBuildFunction runs a post-build callback function registered on a build target via post_build = <...>.
//
// This is called after the target has been built and it is given the combined stdout/stderr of
// the build process. This output is passed to the post-build Python function which can then
// generate new targets or add dependencies to existing unbuilt targets.
func RunPostBuildFunction(tid int, state *core.BuildState, target *core.BuildTarget, out string) error {
	state.LogBuildResult(tid, target.Label, core.PackageParsing,
		fmt.Sprintf("Running post-build function for %s", target.Label))
	pkg := state.Graph.Package(target.Label.PackageName)
	pkg.BuildCallbackMutex.Lock()
	defer pkg.BuildCallbackMutex.Unlock()
	log.Debug("Running post-build function for %s. Build output:\n%s\n", target.Label, out)
	if err := runPostBuildFunction(pkg, target, out); err != nil {
		state.LogBuildError(tid, target.Label, core.ParseFailed, err, "Failed post-build function for %s", target.Label)
		return err
	}
	rescanDeps(state, pkg)
	state.LogBuildResult(tid, target.Label, core.TargetBuilding,
		fmt.Sprintf("Finished post-build function for %s", target.Label))
	return nil
}

func rescanDeps(state *core.BuildState, pkg *core.Package) {
	// Run over all the targets in this package and ensure that any newly added dependencies enter the build queue.
	for _, target := range pkg.Targets {
		// TODO(pebers): this is pretty brutal; we're forcing a recheck of all dependencies
		//               in case we have any new targets. It'd be better to do it only for
		//               targets that need it but it's not easy to tell we're in a post build
		//               function at the point we'd need to do that.
		if !state.Graph.AllDependenciesResolved(target) {
			for _, dep := range target.DeclaredDependencies() {
				state.Graph.AddDependency(target.Label, dep)
			}
		}
		s := target.State()
		if s < core.Built && s > core.Inactive {
			addDep(state, target.Label, core.OriginalTarget, true, false)
		}
	}
}
