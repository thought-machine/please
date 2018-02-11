// Package parse implements parsing of the BUILD files via an embedded Python interpreter.
//
// The actual work here is done by an embedded PyPy instance. Various rules are built in to
// the binary itself using go-bindata to embed the .py files; these are always available to
// all programs which is rather nice, but it does mean that must be run before 'go run' etc
// will work as expected.
package parse

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sync"

	"core"
	"parse/asp"
)

// Parse parses the package corresponding to a single build label. The label can be :all to add all targets in a package.
// It is not an error if the package has already been parsed.
//
// By default, after the package is parsed, any targets that are now needed for the build and ready
// to be built are queued, and any new packages are queued for parsing. When a specific label is requested
// this is straightforward, but when parsing for pseudo-targets like :all and ..., various flags affect it:
// If 'noDeps' is true, then no new packages will be added and no new targets queued.
// 'include' and 'exclude' refer to the labels of targets to be added. If 'include' is non-empty then only
// targets with at least one matching label are added. Any targets with a label in 'exclude' are not added.
// 'forSubinclude' is set when the parse is required for a subinclude target so should proceed
// even when we're not otherwise building targets.
func Parse(tid int, state *core.BuildState, label, dependor core.BuildLabel, noDeps bool, include, exclude []string, forSubinclude bool) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = fmt.Errorf("%s", r)
			}
			state.LogBuildError(tid, label, core.ParseFailed, err, "Failed to parse package")
		}
	}()
	// First see if this package already exists; once it's in the graph it will have been parsed.
	pkg := state.Graph.Package(label.PackageName)
	if pkg != nil {
		// Does exist, all we need to do is toggle on this target
		activateTarget(state, pkg, label, dependor, noDeps, forSubinclude, include, exclude)
		return
	}
	// Check whether this guy exists within a subrepo. If so we will need to make sure that's available first.
	if subrepo := state.Graph.SubrepoFor(label.PackageName); subrepo != nil && subrepo.Target != nil {
		if deferParse(subrepo.Target.Label, label.PackageName) {
			log.Debug("Deferring parse of %s pending subrepo dependency %s", label, subrepo.Target.Label)
			return
		}
	}

	// We use the name here to signal undeferring of a package. If we get that we need to retry the package regardless.
	if dependor.Name != "_UNDEFER_" && !firstToParse(label, dependor) {
		// Check this again to avoid a potential race
		if pkg = state.Graph.Package(label.PackageName); pkg != nil {
			activateTarget(state, pkg, label, dependor, noDeps, forSubinclude, include, exclude)
		} else if forSubinclude {
			// Need to make sure this guy happens, so re-add him to the queue.
			// It should be essentially idempotent but we need to make sure that the task with
			// forSubinclude = true is processed at some point, not just ones where it's false.
			log.Debug("Re-adding pending parse for %s", label)
			core.State.AddPendingParse(label, dependor, true)
		} else {
			log.Debug("Skipping pending parse for %s", label)
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
	log.Debug("Retrieved %d pending targets for %s", len(pending), label)
	pendingTargetMutex.Unlock() // Nothing will look up this package in the map again.
	for targetName, dependors := range pending {
		for _, dependor := range dependors {
			log.Debug("Undeferring pending target %s now we've got %s", dependor, targetName)
			lbl := core.BuildLabel{PackageName: label.PackageName, Name: targetName}
			activateTarget(state, pkg, lbl, dependor, noDeps, forSubinclude, include, exclude)
		}
	}
	state.LogBuildResult(tid, label, core.PackageParsed, "Parsed")
}

// PrintRuleArgs prints the arguments of all builtin rules (plus any associated ones from the given targets)
func PrintRuleArgs(state *core.BuildState, labels []core.BuildLabel) {
	p := newAspParser(state)
	for _, l := range labels {
		t := state.Graph.TargetOrDie(l)
		for _, out := range t.Outputs() {
			p.MustLoadBuiltins(path.Join(t.OutDir(), out), nil, nil)
		}
	}
	b, err := json.MarshalIndent(p.Environment(), "", "  ")
	if err != nil {
		log.Fatalf("Failed JSON encoding: %s", err)
	}
	os.Stdout.Write(b)
}

// activateTarget marks a target as active (ie. to be built) and adds its dependencies as pending parses.
func activateTarget(state *core.BuildState, pkg *core.Package, label, dependor core.BuildLabel, noDeps, forSubinclude bool, include, exclude []string) {
	if !label.IsAllTargets() && state.Graph.Target(label) == nil {
		msg := fmt.Sprintf("Parsed build file %s but it doesn't contain target %s", pkg.Filename, label.Name)
		if dependor != core.OriginalTarget {
			msg += fmt.Sprintf(" (depended on by %s)", dependor)
		}
		panic(msg + suggestTargets(pkg, label, dependor))
	}
	if noDeps && !dependor.IsAllTargets() { // IsAllTargets indicates requirement for parse
		return // Some kinds of query don't need a full recursive parse.
	} else if label.IsAllTargets() {
		for _, target := range pkg.AllTargets() {
			// Don't activate targets that were added in a post-build function; that causes a race condition
			// between the post-build functions running and other things trying to activate them too early.
			if target.ShouldInclude(include, exclude) && !target.AddedPostBuild {
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
			addDep(state, l, dependor, false, forSubinclude || dependor.IsAllTargets())
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
	pendingTargets[label.PackageName] = map[string][]core.BuildLabel{label.Name: {dependor}}
	return true
}

// deferParse defers the parsing of a package until the given label has been built.
// Returns true if it was deferred, or false if it's already built.
func deferParse(label core.BuildLabel, pkgName string) bool {
	pendingTargetMutex.Lock()
	defer pendingTargetMutex.Unlock()
	if target := core.State.Graph.Target(label); target != nil && target.State() >= core.Built {
		return false
	}
	log.Debug("Deferring parse of %s pending %s", pkgName, label)
	if m, present := deferredParses[label.PackageName]; present {
		m[label.Name] = append(m[label.Name], pkgName)
	} else {
		deferredParses[label.PackageName] = map[string][]string{label.Name: {pkgName}}
	}
	log.Debug("Adding pending parse for %s", label)
	core.State.AddPendingParse(label, core.BuildLabel{PackageName: pkgName, Name: "all"}, true)
	return true
}

// undeferAnyParses un-defers the parsing of a package if it depended on some subinclude target being built.
func undeferAnyParses(state *core.BuildState, target *core.BuildTarget) {
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

	if err := state.Parser.ParseFile(state, pkg, pkg.Filename); err == errDeferParse {
		return nil // Indicates deferral
	} else if required, l := asp.RequiresSubinclude(err); required {
		if deferParse(l, pkg.Name) {
			return nil // similarly, deferral
		}
		// If we get here, the target wasn't available to subinclude before, but is now. Try it again.
		return parsePackage(state, label, dependor)
	} else if err != nil {
		panic(err) // TODO(peterebden): Should just return this...
	}

	allTargets := pkg.AllTargets()
	for _, target := range allTargets {
		state.Graph.AddTarget(target)
		if target.IsFilegroup {
			// At least register these guys as outputs.
			// It's difficult to handle non-file sources because we don't know if they're
			// parsed yet - recall filegroups are a special case for this since they don't
			// explicitly declare their outputs but can re-output other rules' outputs.
			for _, src := range target.AllLocalSources() {
				pkg.MustRegisterOutput(src, target)
			}
		} else {
			for _, out := range target.DeclaredOutputs() {
				pkg.MustRegisterOutput(out, target)
			}
			for _, out := range target.TestOutputs {
				if !core.IsGlob(out) {
					pkg.MustRegisterOutput(out, target)
				}
			}
		}
	}
	// Do this in a separate loop so we get intra-package dependencies right now.
	for _, target := range allTargets {
		for _, dep := range target.DeclaredDependencies() {
			state.Graph.AddDependency(target.Label, dep)
		}
	}
	// Verify some details of the output files in the background. Don't need to wait for this
	// since it only issues warnings sometimes.
	go pkg.VerifyOutputs()
	state.Graph.AddPackage(pkg) // Calling this means nobody else will add entries to pendingTargets for this package.
	return pkg
}

func buildFileName(state *core.BuildState, pkgName string) string {
	// Bazel defines targets in its "external" package from its WORKSPACE file.
	// We will fake this by treating that as an actual package file...
	// TODO(peterebden): They may be moving away from their "external" nomenclature?
	if state.Config.Bazel.Compatibility && pkgName == "external" || pkgName == "workspace" {
		return "WORKSPACE"
	}
	for _, buildFileName := range state.Config.Parse.BuildFileName {
		if filename := path.Join(pkgName, buildFileName); core.FileExists(filename) {
			return filename
		}
	}
	// Could be a subrepo...
	if subrepo := state.Graph.SubrepoFor(pkgName); subrepo != nil {
		return buildFileName(state, subrepo.Dir(pkgName))
	}
	return ""
}

// Adds a single target to the build queue.
func addDep(state *core.BuildState, label, dependor core.BuildLabel, rescan, forceBuild bool) {
	// Stop at any package that's not loaded yet
	if state.Graph.Package(label.PackageName) == nil {
		if forceBuild {
			log.Debug("Adding forced pending parse of %s", label)
		}
		state.AddPendingParse(label, dependor, forceBuild)
		return
	}
	target := state.Graph.Target(label)
	if target == nil {
		log.Fatalf("Target %s (referenced by %s) doesn't exist\n", label, dependor)
	}
	if forceBuild {
		log.Debug("Forcing build of %s", label)
	}
	if target.State() >= core.Active && !rescan && !forceBuild {
		return // Target is already tagged to be built and likely on the queue.
	}
	// Only do this bit if we actually need to build the target
	if !target.SyncUpdateState(core.Inactive, core.Semiactive) && !rescan && !forceBuild {
		return
	}
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
		if forceBuild {
			log.Debug("Forcing build of dep %s -> %s", label, dep)
		}
		addDep(state, dep, label, false, forceBuild)
	}
}

func rescanDeps(state *core.BuildState, changed map[*core.BuildTarget]struct{}) {
	// Run over all the changed targets in this package and ensure that any newly added dependencies enter the build queue.
	for target := range changed {
		if !state.Graph.AllDependenciesResolved(target) {
			for _, dep := range target.DeclaredDependencies() {
				state.Graph.AddDependency(target.Label, dep)
			}
		}
		if s := target.State(); s < core.Built && s > core.Inactive {
			addDep(state, target.Label, core.OriginalTarget, true, false)
		}
	}
}
