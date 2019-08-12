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
	"strings"

	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/worker"
)

var log = logging.MustGetLogger("parse")

// Parse parses the package corresponding to a single build label. The label can be :all to add all targets in a package.
// It is not an error if the package has already been parsed.
//
// By default, after the package is parsed, any targets that are now needed for the build and ready
// to be built are queued, and any new packages are queued for parsing. When a specific label is requested
// this is straightforward, but when parsing for pseudo-targets like :all and ..., various flags affect it:
// 'include' and 'exclude' refer to the labels of targets to be added. If 'include' is non-empty then only
// targets with at least one matching label are added. Any targets with a label in 'exclude' are not added.
// 'forSubinclude' is set when the parse is required for a subinclude target so should proceed
// even when we're not otherwise building targets.
func Parse(tid int, state *core.BuildState, label, dependent core.BuildLabel, forSubinclude bool) {
	if err := parse(tid, state, label, dependent, forSubinclude); err != nil {
		state.LogBuildError(tid, label, core.ParseFailed, err, "Failed to parse package")
	}
}

func parse(tid int, state *core.BuildState, label, dependent core.BuildLabel, forSubinclude bool) error {
	// See if something else has parsed this package first.
	pkg := state.WaitForPackage(label)
	if pkg != nil {
		// Does exist, all we need to do is toggle on this target
		return activateTarget(tid, state, pkg, label, dependent, forSubinclude)
	}
	// If we get here then it falls to us to parse this package.
	state.LogBuildResult(tid, label, core.PackageParsing, "Parsing...")

	subrepo, err := checkSubrepo(tid, state, label, dependent)
	if err != nil {
		return err
	} else if subrepo != nil && subrepo.Target != nil {
		// We have got the definition of the subrepo but it depends on something, make sure that has been built.
		state.WaitForBuiltTarget(subrepo.Target.Label, label)
	}
	// Subrepo & nothing else means we just want to ensure that subrepo is present.
	if label.Subrepo != "" && label.PackageName == "" && label.Name == "" {
		return nil
	}
	pkg, err = parsePackage(state, label, dependent, subrepo)
	if err != nil {
		return err
	}
	state.LogBuildResult(tid, label, core.PackageParsed, "Parsed package")
	return activateTarget(tid, state, pkg, label, dependent, forSubinclude)
}

// checkSubrepo checks whether this guy exists within a subrepo. If so we will need to make sure that's available first.
func checkSubrepo(tid int, state *core.BuildState, label, dependent core.BuildLabel) (*core.Subrepo, error) {
	if label.Subrepo == "" {
		return nil, nil
	} else if subrepo := state.Graph.Subrepo(label.Subrepo); subrepo != nil {
		return subrepo, nil
	}
	// We don't have the definition of it at all. Need to parse that first.
	sl := label.SubrepoLabel()
	if handled, err := parseSubrepoPackage(tid, state, sl.PackageName, "", label); err != nil {
		return nil, err
	} else if !handled {
		if _, err := parseSubrepoPackage(tid, state, sl.PackageName, dependent.Subrepo, label); err != nil {
			return nil, err
		}
	}
	if subrepo := state.Graph.Subrepo(label.Subrepo); subrepo != nil {
		return subrepo, nil
	} else if subrepo := checkArchSubrepo(state, label.Subrepo); subrepo != nil {
		return subrepo, nil
	}
	// Fix for #577; fallback like above, it might be defined within the subrepo.
	if handled, err := parseSubrepoPackage(tid, state, sl.PackageName, dependent.Subrepo, label); handled && err == nil {
		return state.Graph.Subrepo(label.Subrepo), nil
	}
	return nil, fmt.Errorf("Subrepo %s is not defined", label.Subrepo)
}

// parseSubrepoPackage parses a package to make sure subrepos are available.
func parseSubrepoPackage(tid int, state *core.BuildState, pkg, subrepo string, dependent core.BuildLabel) (bool, error) {
	if state.Graph.Package(pkg, subrepo) == nil {
		// Don't have it already, must parse.
		label := core.BuildLabel{Subrepo: subrepo, PackageName: pkg, Name: "all"}
		return true, parse(tid, state, label, dependent, true)
	}
	return false, nil
}

// checkArchSubrepo checks if a target refers to a cross-compiling subrepo.
// Those don't have to be explicitly defined - maybe we should insist on that, but it's nicer not to have to.
func checkArchSubrepo(state *core.BuildState, name string) *core.Subrepo {
	var arch cli.Arch
	if err := arch.UnmarshalFlag(name); err == nil {
		return state.Graph.MaybeAddSubrepo(core.SubrepoForArch(state, arch))
	}
	return nil
}

// activateTarget marks a target as active (ie. to be built) and adds its dependencies as pending parses.
func activateTarget(tid int, state *core.BuildState, pkg *core.Package, label, dependent core.BuildLabel, forSubinclude bool) error {
	if !label.IsAllTargets() && state.Graph.Target(label) == nil {
		if label.Subrepo == "" && label.PackageName == "" && label.Name == dependent.Subrepo {
			if subrepo := checkArchSubrepo(state, label.Name); subrepo != nil {
				state.LogBuildResult(tid, label, core.TargetBuilt, "Instantiated subrepo")
				return nil
			}
		}
		if state.Config.Bazel.Compatibility && forSubinclude {
			// Bazel allows some things that look like build targets but aren't - notably the syntax
			// to load(). It suits us to treat that as though it is one, but we now have to
			// implicitly make it available.
			exportFile(state, pkg, label)
		} else {
			msg := fmt.Sprintf("Parsed build file %s but it doesn't contain target %s", pkg.Filename, label.Name)
			if dependent != core.OriginalTarget {
				msg += fmt.Sprintf(" (depended on by %s)", dependent)
			}
			return fmt.Errorf(msg + suggestTargets(pkg, label, dependent))
		}
	}
	if state.ParsePackageOnly && !forSubinclude {
		return nil // Some kinds of query don't need a full recursive parse.
	} else if label.IsAllTargets() {
		if dependent == core.OriginalTarget {
			for _, target := range pkg.AllTargets() {
				// Don't activate targets that were added in a post-build function; that causes a race condition
				// between the post-build functions running and other things trying to activate them too early.
				if state.ShouldInclude(target) && !target.AddedPostBuild {
					// Must always do this for coverage because we need to calculate sources of
					// non-test targets later on.
					if !state.NeedTests || target.IsTest || state.NeedCoverage {
						if err := state.QueueTarget(target.Label, dependent, false, dependent.IsAllTargets()); err != nil {
							return err
						}
					}
				}
			}
		}
	} else {
		for _, l := range state.Graph.DependentTargets(dependent, label) {
			// We use :all to indicate a dependency needed for parse.
			if err := state.QueueTarget(l, dependent, false, forSubinclude || dependent.IsAllTargets()); err != nil {
				return err
			}
		}
	}
	return nil
}

// parsePackage performs the initial parse of a package.
func parsePackage(state *core.BuildState, label, dependent core.BuildLabel, subrepo *core.Subrepo) (*core.Package, error) {
	packageName := label.PackageName
	pkg := core.NewPackage(packageName)
	pkg.Subrepo = subrepo
	if subrepo != nil {
		pkg.SubrepoName = subrepo.Name
	}
	filename, dir := buildFileName(state, label.PackageName, subrepo)
	if filename == "" {
		if success, err := providePackage(state, pkg); err != nil {
			return nil, err
		} else if !success && packageName == "" && dependent.Subrepo == "pleasings" && subrepo == nil && state.Config.Parse.BuiltinPleasings {
			// Deliberate fallthrough, for the case where someone depended on the default
			// @pleasings subrepo, and there is no BUILD file at their root.
		} else if !success {
			exists := core.PathExists(dir)
			// Handle quite a few cases to provide more obvious error messages.
			if dependent != core.OriginalTarget && exists {
				return nil, fmt.Errorf("%s depends on %s, but there's no %s file in %s/", dependent, label, buildFileNames(state.Config.Parse.BuildFileName), dir)
			} else if dependent != core.OriginalTarget {
				return nil, fmt.Errorf("%s depends on %s, but the directory %s doesn't exist", dependent, label, dir)
			} else if exists {
				return nil, fmt.Errorf("Can't build %s; there's no %s file in %s/", label, buildFileNames(state.Config.Parse.BuildFileName), dir)
			}
			return nil, fmt.Errorf("Can't build %s; the directory %s doesn't exist", label, dir)
		}
	} else {
		pkg.Filename = filename
		if err := state.Parser.ParseFile(state, pkg, pkg.Filename); err != nil {
			return nil, err
		}
	}
	// If the config setting is on, we "magically" register a default repo called @pleasings.
	if packageName == "" && subrepo == nil && state.Config.Parse.BuiltinPleasings && pkg.Target("pleasings") == nil {
		if _, err := state.Parser.(*aspParser).asp.ParseReader(pkg, strings.NewReader(pleasings)); err != nil {
			log.Fatalf("Failed to load pleasings: %s", err) // This shouldn't happen, of course.
		}
	}
	// Verify some details of the output files in the background. Don't need to wait for this
	// since it only issues warnings sometimes.
	go pkg.VerifyOutputs()
	state.Graph.AddPackage(pkg) // Calling this means nobody else will add entries to pendingTargets for this package.
	return pkg, nil
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
		if filename := path.Join(core.RepoRoot, pkgName, buildFileName); fs.FileExists(filename) {
			return filename, pkgName
		}
	}
	return "", pkgName
}

func rescanDeps(state *core.BuildState, changed map[*core.BuildTarget]struct{}) error {
	// Run over all the changed targets in this package and ensure that any newly added dependencies enter the build queue.
	for target := range changed {
		if !state.Graph.AllDependenciesResolved(target) {
			for _, dep := range target.DeclaredDependencies() {
				state.Graph.AddDependency(target.Label, dep)
			}
		}
		if s := target.State(); s < core.Built && s > core.Inactive {
			if err := state.QueueTarget(target.Label, core.OriginalTarget, true, false); err != nil {
				return err
			}
		}
	}
	return nil
}

// This is the builtin subrepo for pleasings.
// TODO(peterebden): Should really provide a github_archive builtin that knows how to construct
//                   the URL and strip_prefix etc.
const pleasings = `
http_archive(
    name = "pleasings",
    strip_prefix = "pleasings-master",
    urls = ["https://github.com/thought-machine/pleasings/archive/master.zip"],
)
`

// providePackage looks through all the configured BUILD file providers to see if any of them
// can handle the given package. It returns true if any of them did.
// N.B. More than one is allowed to handle a single directory.
func providePackage(state *core.BuildState, pkg *core.Package) (bool, error) {
	if len(state.Config.Provider) == 0 {
		return false, nil
	}
	success := false
	label := pkg.Label()
	for name, p := range state.Config.Provider {
		if !shouldProvide(p.Path, label) {
			continue
		}
		t := state.WaitForBuiltTarget(p.Target, label)
		outs := t.Outputs()
		if !t.IsBinary && len(outs) != 1 {
			log.Error("Cannot use %s as build provider %s, it must be a binary with exactly 1 output.", p.Target, name)
			continue
		}
		dir := pkg.SourceRoot()
		resp, err := worker.ProvideParse(state, path.Join(t.OutDir(), outs[0]), dir)
		if err != nil {
			return false, fmt.Errorf("Failed to start build provider %s: %s", name, err)
		} else if resp != "" {
			log.Debug("Received BUILD file from %s provider for %s: %s", name, dir, resp)
			if err := state.Parser.ParseReader(state, pkg, strings.NewReader(resp)); err != nil {
				return false, err
			}
			success = true
		}
	}
	return success, nil
}

// shouldProvide returns true if a provider's set of configured paths overlaps a package.
func shouldProvide(paths []core.BuildLabel, label core.BuildLabel) bool {
	for _, p := range paths {
		if p.Includes(label) {
			return true
		}
	}
	return false
}

// exportFile adds a single-file export target. This is primarily used for Bazel compat.
func exportFile(state *core.BuildState, pkg *core.Package, label core.BuildLabel) {
	t := core.NewBuildTarget(label)
	t.Subrepo = pkg.Subrepo
	t.IsFilegroup = true
	t.AddSource(core.NewFileLabel(label.Name, pkg))
	state.AddTarget(pkg, t)
}
