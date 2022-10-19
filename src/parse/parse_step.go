// Package parse implements handling parse tasks for BUILD files.
//
// The actual work to interpret them is done by the //src/parse/asp package; this
// package handles requests for parsing build targets and triggering them to
// start building when ready.
package parse

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

var log = logging.Log

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
	if !forSubinclude {
		state.Parser.WaitForInit()
	}
	subrepo, err := checkSubrepo(tid, state, label, dependent, forSubinclude)
	if err != nil {
		return err
	}

	// See if something else has parsed this package first.
	pkg := state.SyncParsePackage(label)
	if pkg != nil {
		// Does exist, all we need to do is toggle on this target
		return activateTarget(state, pkg, label, dependent, forSubinclude)
	}
	// If we get here then it falls to us to parse this package.
	state.LogParseResult(tid, label, core.PackageParsing, "Parsing...")

	if subrepo != nil && subrepo.Target != nil {
		// We have got the definition of the subrepo but it depends on something, make sure that has been built.
		state.WaitForTargetAndEnsureDownload(subrepo.Target.Label, label)
		if err := subrepo.LoadSubrepoConfig(); err != nil {
			return err
		}
	}

	if subrepo != nil {
		state = subrepo.State
		if !forSubinclude {
			state.Parser.WaitForInit()
		}
	}

	// Subrepo & nothing else means we just want to ensure that subrepo is present.
	if label.Subrepo != "" && label.PackageName == "" && label.Name == "" {
		return nil
	}
	pkg, err = parsePackage(state, label, dependent, subrepo)

	if err != nil {
		return err
	}
	state.LogParseResult(tid, label, core.PackageParsed, "Parsed package")
	return activateTarget(state, pkg, label, dependent, forSubinclude)
}

// checkSubrepo checks whether this guy exists within a subrepo. If so we will need to make sure that's available first.
func checkSubrepo(tid int, state *core.BuildState, label, dependent core.BuildLabel, forSubinclude bool) (*core.Subrepo, error) {
	if label.Subrepo == "" {
		return nil, nil
	} else if subrepo := state.Graph.Subrepo(label.Subrepo); subrepo != nil {
		return subrepo, nil
	}

	sl := label.SubrepoLabel(state, dependent.Subrepo)

	// Local subincludes are when we subinclude from a subrepo defined in the current package
	localSubinclude := label.Subrepo == dependent.Subrepo && label.PackageName == dependent.PackageName && forSubinclude

	// If we're including from the same package, we don't want to parse the subrepo package
	if !localSubinclude {
		if handled, err := parseSubrepoPackage(tid, state, sl.PackageName, "", label); err != nil {
			return nil, err
		} else if !handled {
			// They may have meant a subrepo that was defined in the dependent label's subrepo rather than the host
			// repo
			if _, err := parseSubrepoPackage(tid, state, sl.PackageName, dependent.Subrepo, label); err != nil {
				return nil, err
			}
		}
	}
	if subrepo := state.Graph.Subrepo(label.Subrepo); subrepo != nil {
		return subrepo, nil
	} else if subrepo := checkArchSubrepo(state, label.Subrepo); subrepo != nil {
		return subrepo, nil
	}
	if !localSubinclude {
		// Fix for #577; fallback like above, it might be defined within the subrepo.
		if handled, err := parseSubrepoPackage(tid, state, sl.PackageName, dependent.Subrepo, label); handled && err == nil {
			return state.Graph.Subrepo(label.Subrepo), nil
		}
		return nil, fmt.Errorf("Subrepo %s is not defined (referenced by %s)", label.Subrepo, dependent)
	}
	// For local subincludes, the subrepo has to already be defined at this point in the BUILD file
	return nil, fmt.Errorf("Subrepo %v is not defined yet. It must appear before it is used by subinclude()", sl)
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
func activateTarget(state *core.BuildState, pkg *core.Package, label, dependent core.BuildLabel, forSubinclude bool) error {
	if !label.IsAllTargets() && state.Graph.Target(label) == nil {
		if label.Subrepo == "" && label.PackageName == "" && label.Name == dependent.Subrepo {
			if subrepo := checkArchSubrepo(state, label.Name); subrepo != nil {
				state.ArchSubrepoInitialised(label)
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
					if !state.NeedTests || target.IsTest() || state.NeedCoverage {
						if err := state.QueueTarget(target.Label, dependent, dependent.IsAllTargets()); err != nil {
							return err
						}
					}
				}
			}
		}
	} else {
		for _, l := range state.Graph.DependentTargets(dependent, label) {
			// We use :all to indicate a dependency needed for parse.
			if err := state.QueueTarget(l, dependent, forSubinclude || dependent.IsAllTargets()); err != nil {
				return err
			}
		}
	}
	return nil
}

// parsePackage parses a BUILD file and adds the package to the build graph
func parsePackage(state *core.BuildState, label, dependent core.BuildLabel, subrepo *core.Subrepo) (*core.Package, error) {
	packageName := label.PackageName
	pkg := core.NewPackage(packageName)
	pkg.Subrepo = subrepo
	if subrepo != nil {
		pkg.SubrepoName = subrepo.Name
	}
	// Only load the internal package for the host repo's state
	if state.ParentState == nil && packageName == InternalPackageName {
		pkgStr, err := GetInternalPackage(state.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to generate internal package: %w", err)
		}
		if err := state.Parser.ParseReader(pkg, strings.NewReader(pkgStr)); err != nil {
			return nil, fmt.Errorf("failed to parse internal package: %w", err)
		}
	} else {
		filename, dir := buildFileName(state, label.PackageName, subrepo)
		if filename != "" {
			pkg.Filename = filename
			if err := state.Parser.ParseFile(pkg, pkg.Filename); err != nil {
				return nil, err
			}
		} else {
			exists := core.PathExists(dir)
			// Handle quite a few cases to provide more obvious error messages.
			if dependent != core.OriginalTarget && exists {
				return nil, fmt.Errorf("%s depends on %s, but there's no %s file in %s/", dependent, label, buildFileNames(state.Config.Parse.BuildFileName), dir)
			} else if dependent != core.OriginalTarget {
				return nil, fmt.Errorf("%s depends on %s, but the directory %s doesn't exist: %s", dependent, label, dir, packageName)
			} else if exists {
				return nil, fmt.Errorf("Can't build %s; there's no %s file in %s/", label, buildFileNames(state.Config.Parse.BuildFileName), dir)
			}
			return nil, fmt.Errorf("Can't build %s; the directory %s doesn't exist", label, dir)
		}
	}

	// Verifies some details of the output files. This can only be perfomed after the whole package has been parsed as
	// it guarantees that all necessary information between targets has been retrieved.
	if state.Config.FeatureFlags.PackageOutputsStrictness {
		go pkg.MustVerifyOutputs()
	} else {
		go pkg.VerifyOutputs()
	}

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
		if filename := filepath.Join(pkgName, buildFileName); fs.FileExists(filename) {
			return filename, pkgName
		}
	}
	return "", pkgName
}

// exportFile adds a single-file export target. This is primarily used for Bazel compat.
func exportFile(state *core.BuildState, pkg *core.Package, label core.BuildLabel) {
	t := core.NewBuildTarget(label)
	t.Subrepo = pkg.Subrepo
	t.IsFilegroup = true
	t.AddSource(core.NewFileLabel(label.Name, pkg))
	state.AddTarget(pkg, t)
}
