// Package parse implements handling parse tasks for BUILD files.
//
// The actual work to interpret them is done by the //src/parse/asp package; this
// package handles requests for parsing build targets and triggering them to
// start building when ready.
package parse

import (
	"errors"
	"fmt"
	"io"
	iofs "io/fs"
	"path/filepath"
	"strings"

	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

var log = logging.Log

var ErrMissingPackage = errors.New("missing package")

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
func Parse(state *core.BuildState, label, dependent core.BuildLabel, mode core.ParseMode) {
	if err := parse(state, label, dependent, mode); err != nil {
		state.LogBuildError(label, core.ParseFailed, err, "Failed to parse package")
	}
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
		// Does exist, all we need to do is toggle on this target
		return state.ActivateTarget(pkg, label, dependent, mode)
	}
	// If we get here then it falls to us to parse this package.
	state.LogParseResult(label, core.PackageParsing, "Parsing...")

	if subrepo != nil && subrepo.Target != nil {
		// We have got the definition of the subrepo, but it depends on something, make sure that has been built.
		state.WaitForBuiltTarget(subrepo.Target.Label, label, mode|core.ParseModeForSubinclude)
		if !subrepo.Target.State().IsBuilt() {
			return fmt.Errorf("%v: failed to build subrepo", label)
		}
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

	// The target likely got activated already, however we activate here to handle pseudo-targets (:all), and to let
	// this error when the target doesn't exist.
	return state.ActivateTarget(pkg, label, dependent, mode)
}

func inSamePackage(label, dependent core.BuildLabel) bool {
	return !dependent.IsOriginalTarget() && label.Subrepo == dependent.Subrepo && label.PackageName == dependent.PackageName
}

// checkSubrepo checks if the label we're parsing is within a subrepo, returning that subrepo, if present in the label.
//
// The subrepo target can be inferred from the subrepo name using convention i.e. ///foo/bar//:baz has a subrepo label
// //foo:bar. checkSubrepo parses package foo, expecting a call to `subrepo()` that registers a subrepo named foo/bar,
// so it can return it.
func checkSubrepo(state *core.BuildState, label, dependent core.BuildLabel, mode core.ParseMode) (*core.Subrepo, error) {
	if label.Subrepo == "" {
		return nil, nil
	}

	// Check if we already have it
	if subrepo := state.Graph.Subrepo(label.Subrepo); subrepo != nil {
		return subrepo, nil
	}

	// SubrepoLabel returns the expected build label for the subrepo's target. Parsing its package should give us the
	// subrepo we're looking for.
	sl := label.SubrepoLabel(state)

	// This can happen when we subinclude() a target in a subrepo from the same package the subrepo is defined in. In,
	// this case, the subrepo must be registered by now. We shouldn't continue to try and parse the subrepo package, as
	// it's the current package we're parsing, which would result in a lockup.
	if inSamePackage(sl, dependent) {
		return nil, fmt.Errorf("subrepo %v is not defined in this package yet. It must appear before it is used by %v", sl.Subrepo, dependent)
	}

	// Try parsing the package in the host repo first.
	s, err := parseSubrepoPackage(state, sl.PackageName, sl.Subrepo, label, mode)
	if err != nil || s != nil {
		return s, handleParseSubrepoErr(err, dependent, label)
	}

	// They may have meant a subrepo that was defined in the dependent label's subrepo rather than the host repo
	s, err = parseSubrepoPackage(state, sl.PackageName, dependent.Subrepo, label, mode)
	if err != nil || s != nil {
		return s, handleParseSubrepoErr(err, dependent, label)
	}

	return nil, fmt.Errorf("Subrepo %s is not defined (referenced by %s)", label.Subrepo, dependent)
}

// handleParseSubrepoErr reports a more sensible error when we go to parse a package in a subrepo, but the subrepo
// package doesn't exist.
func handleParseSubrepoErr(err error, dependant, label core.BuildLabel) error {
	if errors.Is(err, ErrMissingPackage) {
		log.Debugf("missing subrepo package: %v", err)
		return fmt.Errorf("%v depends on %v but the subrepo doesn't exist", dependant, label)
	}
	return err
}

// parseSubrepoPackage parses a package to make sure subrepos are available.
func parseSubrepoPackage(state *core.BuildState, subrepoPkg, subrepoSubrepo string, dependent core.BuildLabel, mode core.ParseMode) (*core.Subrepo, error) {
	// Check if the subrepo package exists
	if state.Graph.Package(subrepoPkg, subrepoSubrepo) == nil {
		// Don't have it already, must parse.
		label := core.BuildLabel{Subrepo: subrepoSubrepo, PackageName: subrepoPkg, Name: "all"}
		if err := parse(state, label, dependent, mode|core.ParseModeForSubinclude); err != nil {
			return nil, err
		}
	}

	// Now that we know its package is parsed, we expect the subrepo to be registered
	//
	// NB: even if we didn't parse it above, this might've been parsed by a different thread. Check again to avoid nasty
	// race conditions.
	s := state.Graph.Subrepo(dependent.Subrepo)
	if s != nil {
		return s, nil
	}

	// If it hasn't, perhaps the subrepo is an architecture subrepo
	return state.CheckArchSubrepo(dependent.Subrepo), nil
}

func openFile(fs iofs.FS, subrepoName, name string) (io.ReadSeekCloser, error) {
	file, err := fs.Open(name)
	if err != nil {
		return nil, fmt.Errorf("failed to open build file: %v", err)
	}

	reader, ok := file.(io.ReadSeekCloser)
	if !ok {
		return nil, fmt.Errorf("opened file is not seekable: ///%v/%v", subrepoName, name)
	}
	return reader, nil
}

// parsePackage parses a BUILD file and adds the package to the build graph
func parsePackage(state *core.BuildState, label, dependent core.BuildLabel, subrepo *core.Subrepo, mode core.ParseMode) (*core.Package, error) {
	packageName := label.PackageName
	pkg := core.NewPackage(packageName)
	pkg.Subrepo = subrepo
	var fileSystem iofs.FS = fs.HostFS
	if subrepo != nil {
		pkg.SubrepoName = subrepo.Name
		fileSystem = subrepo.FS()
	}
	// Only load the internal package for the host repo's state
	if state.ParentState == nil && packageName == InternalPackageName {
		pkgStr, err := GetInternalPackage(state.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to generate internal package: %w", err)
		}
		if err := state.Parser.ParseReader(pkg, strings.NewReader(pkgStr), &label, &dependent, mode); err != nil {
			return nil, fmt.Errorf("failed to parse internal package: %w", err)
		}
	} else {
		filename, dir := buildFileName(state, subrepo, fileSystem, label.PackageName)
		if filename != "" {
			file, err := openFile(fileSystem, pkg.SubrepoName, filename)
			if err != nil {
				return nil, err
			}
			defer file.Close()

			pkg.Filename = filename
			if err := state.Parser.ParseReader(pkg, file, &label, &dependent, mode); err != nil {
				return nil, err
			}
		} else {
			exists := core.PathExists(dir)
			// Handle quite a few cases to provide more obvious error messages.
			if dependent != core.OriginalTarget && exists {
				return nil, fmt.Errorf("%w: %s depends on %s, but there's no %s file in %s/", ErrMissingPackage, dependent, label, buildFileNames(state.Config.Parse.BuildFileName), dir)
			} else if dependent != core.OriginalTarget {
				return nil, fmt.Errorf("%w: %s depends on %s, but the directory %s doesn't exist: %s", ErrMissingPackage, dependent, label, dir, packageName)
			} else if exists {
				return nil, fmt.Errorf("%w: Can't build %s; there's no %s file in %s/", ErrMissingPackage, label, buildFileNames(state.Config.Parse.BuildFileName), dir)
			}
			return nil, fmt.Errorf("%w: Can't build %s; the directory %s doesn't exist", ErrMissingPackage, label, dir)
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
func buildFileName(state *core.BuildState, subrepo *core.Subrepo, fs iofs.FS, pkgName string) (string, string) {
	config := state.Config
	if subrepo != nil {
		config = subrepo.State.Config
	}
	// Bazel defines targets in its "external" package from its WORKSPACE file.
	// We will fake this by treating that as an actual package file...
	// TODO(peterebden): They may be moving away from their "external" nomenclature?
	if state.Config.Bazel.Compatibility && pkgName == "external" || pkgName == "workspace" {
		return "WORKSPACE", ""
	}
	for _, buildFileName := range config.Parse.BuildFileName {
		filename := filepath.Join(pkgName, buildFileName)
		if info, err := iofs.Stat(fs, filename); err == nil && !info.IsDir() {
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
