// Package parse implements parsing of the BUILD files via an embedded Python interpreter.
//
// The actual work here is done by an embedded PyPy instance. Various rules are built in to
// the binary itself using go-bindata to embed the .py files; these are always available to
// all programs which is rather nice, but it does mean that must be run before 'go run' etc
// will work as expected.
package parse

import (
	"fmt"
	"path"

	"gopkg.in/op/go-logging.v1"

	"cli"
	"core"
	"fs"
)

var log = logging.MustGetLogger("parse")

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
	if err := parse(tid, state, label, dependor, noDeps, include, exclude, forSubinclude); err != nil {
		state.LogBuildError(tid, label, core.ParseFailed, err, "Failed to parse package")
	}
}

func parse(tid int, state *core.BuildState, label, dependor core.BuildLabel, noDeps bool, include, exclude []string, forSubinclude bool) error {
	// See if something else has parsed this package first.
	pkg := state.WaitForPackage(label)
	if pkg != nil {
		// Does exist, all we need to do is toggle on this target
		return activateTarget(state, pkg, label, dependor, noDeps, forSubinclude, include, exclude)
	}
	// If we get here then it falls to us to parse this package.
	state.LogBuildResult(tid, label, core.PackageParsing, "Parsing...")

	subrepo, err := checkSubrepo(tid, state, label, dependor)
	if err != nil {
		return err
	}
	pkg, err = parsePackage(state, label, dependor, subrepo)
	if err != nil {
		return err
	}
	state.LogBuildResult(tid, label, core.PackageParsed, "Parsed package")
	return activateTarget(state, pkg, label, dependor, noDeps, forSubinclude, include, exclude)
}

// checkSubrepo checks whether this guy exists within a subrepo. If so we will need to make sure that's available first.
func checkSubrepo(tid int, state *core.BuildState, label, dependor core.BuildLabel) (*core.Subrepo, error) {
	if label.Subrepo == "" {
		return nil, nil
	}
	subrepo := state.Graph.Subrepo(label.Subrepo)
	if subrepo != nil && subrepo.Target != nil {
		// We have got the definition of the subrepo but it depends on something, make sure that has been built.
		state.WaitForBuiltTarget(subrepo.Target.Label, label)
	} else if subrepo == nil {
		// We don't have the definition of it at all. Need to parse that first.
		sl := label.SubrepoLabel()
		if state.Graph.Package(sl.PackageName, "") != nil {
			// Package has already been parsed and this subrepo doesn't exist.
			if dependor.Subrepo == "" {
				return nil, fmt.Errorf("Subrepo %s is not defined", label.Subrepo)
			}
			// It might be defined in a subrepo itself.
			// This seems mildly dodgy in that it seems to depend a bit much
			// on ordering events but it seems useful to let the top-level repo
			// define the subrepos, since they exist in a global namespace.
			log.Debug("Looking for subrepo %s in subrepo %s", label.Subrepo, dependor.Subrepo)
			sl.Subrepo = dependor.Subrepo
			if state.Graph.Package(sl.PackageName, sl.Subrepo) != nil {
				return nil, fmt.Errorf("Subrepo %s is not defined", label.Subrepo)
			}
		}
		if err := parse(tid, state, sl, label, false, nil, nil, true); err != nil {
			return nil, err
		}
		// Now it's parsed, but might need to wait for it to be built
		return checkSubrepo(tid, state, label, dependor)
	}
	return subrepo, nil
}

// activateTarget marks a target as active (ie. to be built) and adds its dependencies as pending parses.
func activateTarget(state *core.BuildState, pkg *core.Package, label, dependor core.BuildLabel, noDeps, forSubinclude bool, include, exclude []string) error {
	if !label.IsAllTargets() && state.Graph.Target(label) == nil {
		// This might be for cross-compiling in which case it doesn't have to be explicitly
		// specified. Maybe we should insist on that, but it's nicer not to have to.
		if label.PackageName == "" && label.Name == dependor.Subrepo {
			var arch cli.Arch
			if err := arch.UnmarshalFlag(label.Name); err == nil {
				state.Graph.MaybeAddSubrepo(core.SubrepoForArch(state, arch))
				return nil
			}
		}
		msg := fmt.Sprintf("Parsed build file %s but it doesn't contain target %s", pkg.Filename, label.Name)
		if dependor != core.OriginalTarget {
			msg += fmt.Sprintf(" (depended on by %s)", dependor)
		}
		return fmt.Errorf(msg + suggestTargets(pkg, label, dependor))
	}
	if noDeps && !forSubinclude {
		return nil // Some kinds of query don't need a full recursive parse.
	} else if label.IsAllTargets() {
		for _, target := range pkg.AllTargets() {
			// Don't activate targets that were added in a post-build function; that causes a race condition
			// between the post-build functions running and other things trying to activate them too early.
			if state.ShouldInclude(target) && !target.AddedPostBuild {
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
	return nil
}

// parsePackage performs the initial parse of a package.
func parsePackage(state *core.BuildState, label, dependor core.BuildLabel, subrepo *core.Subrepo) (*core.Package, error) {
	packageName := label.PackageName
	pkg := core.NewPackage(packageName)
	pkg.Subrepo = subrepo
	if subrepo != nil {
		pkg.SubrepoName = subrepo.Name
	}
	filename, dir := buildFileName(state, label.PackageName, subrepo)
	if filename == "" {
		exists := core.PathExists(packageName)
		// Handle quite a few cases to provide more obvious error messages.
		if dependor != core.OriginalTarget && exists {
			return nil, fmt.Errorf("%s depends on %s, but there's no BUILD file in %s/", dependor, label, dir)
		} else if dependor != core.OriginalTarget {
			return nil, fmt.Errorf("%s depends on %s, but the directory %s doesn't exist", dependor, label, dir)
		} else if exists {
			return nil, fmt.Errorf("Can't build %s; there's no BUILD file in %s/", label, dir)
		}
		return nil, fmt.Errorf("Can't build %s; the directory %s doesn't exist", label, dir)
	}
	pkg.Filename = filename
	if err := state.Parser.ParseFile(state, pkg, pkg.Filename); err != nil {
		return nil, err
	}
	addPackage(state, pkg)
	return pkg, nil
}

// addPackage adds the given package to the graph, with appropriate dependencies and whatnot.
func addPackage(state *core.BuildState, pkg *core.Package) {
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
				if !fs.IsGlob(out) {
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
}

// buildFileName returns the name of the BUILD file for a package, or the empty string if one
// doesn't exist. It also returns the directory that it looked in.
func buildFileName(state *core.BuildState, pkgName string, subrepo *core.Subrepo) (string, string) {
	config := state.Config
	if subrepo != nil {
		pkgName = subrepo.Dir(pkgName)
		config = subrepo.State.Config
	}
	// Bazel defines targets in its "external" package from its WORKSPACE file.
	// We will fake this by treating that as an actual package file...
	// TODO(peterebden): They may be moving away from their "external" nomenclature?
	if state.Config.Bazel.Compatibility && pkgName == "external" || pkgName == "workspace" {
		return "WORKSPACE", ""
	}
	for _, buildFileName := range config.Parse.BuildFileName {
		if filename := path.Join(pkgName, buildFileName); fs.FileExists(filename) {
			return filename, pkgName
		}
	}
	return "", pkgName
}

// Adds a single target to the build queue.
func addDep(state *core.BuildState, label, dependor core.BuildLabel, rescan, forceBuild bool) {
	// Stop at any package that's not loaded yet
	if state.Graph.PackageByLabel(label) == nil {
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
	if target.State() >= core.Active && !rescan {
		if !forceBuild {
			return // Target is already tagged to be built and likely on the queue.
		}
		log.Debug("Forcing build of %s", label)
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
