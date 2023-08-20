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
func Parse(state *core.BuildState, label, dependent core.BuildLabel, mode core.ParseMode) error {
	if err := parse(state, label, dependent, mode); err != nil {
		state.LogBuildError(label, core.ParseFailed, err, "Failed to parse package")
		return err
	}
	return nil
}

func parse(state *core.BuildState, label, dependent core.BuildLabel, mode core.ParseMode) error {
	subrepo, err := checkSubrepo(state, label, dependent, mode)
	if err != nil {
		return err
	}

	if subrepo != nil {
		state = subrepo.State
	}

	// Ensure that all the preloaded targets are built before we sync the package parse. If we don't do this, we might
	// take the package lock for a package involved in a subinclude, and end up in a deadlock
	if !mode.IsPreload() {
		if err := state.RegisterPreloads(); err != nil {
			return err
		}
	}

	// See if something else has parsed this package first.
	pkg := state.SyncParsePackage(label)
	if pkg != nil {
		// Does exist, nothing else needs doing by us
		return nil
	}
	// If we get here then it falls to us to parse this package.
	state.LogParseResult(label, core.PackageParsing, "Parsing...")

	if subrepo != nil && subrepo.Target != nil {
		// We have got the definition of the subrepo but it depends on something, make sure that has been built.
		state.WaitForTargetAndEnsureDownload(subrepo.Target.Label, label, mode)
		if err := subrepo.State.Initialise(subrepo); err != nil {
			return err
		}
	}

	// Subrepo & nothing else means we just want to ensure that subrepo is present.
	if label.Subrepo != "" && label.PackageName == "" && label.Name == "" {
		return nil
	}
	pkg, err = parsePackage(state, label, dependent, subrepo, mode)

	if err != nil {
		return err
	}
	state.LogParseResult(label, core.PackageParsed, "Parsed package")
	// The target likely got activated already, however we activate here to handle psudotargets (:all), and to let this
	// error when the target doesn't exist.

	return nil
}

// checkSubrepo checks whether this guy exists within a subrepo. If so we will need to make sure that's available first.
func checkSubrepo(state *core.BuildState, label, dependent core.BuildLabel, mode core.ParseMode) (*core.Subrepo, error) {
	if label.Subrepo == "" {
		return nil, nil
	} else if subrepo := state.Graph.Subrepo(label.Subrepo); subrepo != nil {
		return subrepo, nil
	}

	sl := label.SubrepoLabel(state, dependent.Subrepo)

	// Local subincludes are when we subinclude from a subrepo defined in the current package
	localSubinclude := label.Subrepo == dependent.Subrepo && label.PackageName == dependent.PackageName && mode.IsForSubinclude()

	// If we're including from the same package, we don't want to parse the subrepo package. It must already be
	// defined by this point in the file.
	if localSubinclude {
		return nil, fmt.Errorf("Subrepo %v is not defined yet. It must appear before it is used by subinclude()", sl)
	}

	// Try parsing the package in the host repo first.
	s, err := parseSubrepoPackage(state, sl.PackageName, "", label, mode)
	if err != nil || s != nil {
		return s, err
	}

	// They may have meant a subrepo that was defined in the dependent label's subrepo rather than the host
	// repo
	s, err = parseSubrepoPackage(state, sl.PackageName, dependent.Subrepo, label, mode)
	if err != nil || s != nil {
		return s, err
	}

	return nil, fmt.Errorf("Subrepo %s is not defined (referenced by %s)", label.Subrepo, dependent)
}

// parseSubrepoPackage parses a package to make sure subrepos are available.
func parseSubrepoPackage(state *core.BuildState, pkg, subrepo string, dependent core.BuildLabel, mode core.ParseMode) (*core.Subrepo, error) {
	if state.Graph.Package(pkg, subrepo) == nil {
		// Don't have it already, must parse.
		label := core.BuildLabel{Subrepo: subrepo, PackageName: pkg, Name: "all"}
		if err := parse(state, label, dependent, mode|core.ParseModeForSubinclude); err != nil {
			return nil, err
		}
	}

	s := state.Graph.Subrepo(dependent.Subrepo)
	if s != nil {
		// Another thread might've parsed the above package, so we should check if the subrepo has appeared now
		return s, nil
	}

	return state.CheckArchSubrepo(dependent.Subrepo), nil
}

// parsePackage parses a BUILD file and adds the package to the build graph
func parsePackage(state *core.BuildState, label, dependent core.BuildLabel, subrepo *core.Subrepo, mode core.ParseMode) (*core.Package, error) {
	packageName := label.PackageName
	pkg := core.NewPackage(packageName)
	pkg.Subrepo = subrepo
	if subrepo != nil {
		pkg.SubrepoName = subrepo.Name
	}
	if packageName == InternalPackageName {
		pkgStr, err := GetInternalPackage(state.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to generate internal package: %w", err)
		}
		if err := state.Parser.ParseReader(pkg, strings.NewReader(pkgStr), mode); err != nil {
			return nil, fmt.Errorf("failed to parse internal package: %w", err)
		}
	} else {
		filename, dir := buildFileName(state, label.PackageName, subrepo)
		if filename != "" {
			pkg.Filename = filename
			if err := state.Parser.ParseFile(pkg, mode, pkg.Filename); err != nil {
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
	go pkg.MustVerifyOutputs()

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

// buildFileNames returns a descriptive version of the configured BUILD file names.
func buildFileNames(l []string) string {
	if len(l) == 1 {
		return l[0]
	}
	return strings.Join(l[:len(l)-1], ", ") + " or " + l[len(l)-1]
}
