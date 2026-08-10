// Package parse implements handling parse tasks for BUILD files.
//
// The actual work to interpret them is done by the //src/parse/asp package; this
// package handles requests for parsing build targets and triggering them to
// start building when ready.
package parse

import (
	"errors"
	"fmt"
	iofs "io/fs"
	"path/filepath"
	"strings"

	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

var log = logging.Log

var ErrMissingBuildFile = errors.New("build file not found")

// Parse parses the package corresponding to a single build label. The label can be :all to add all targets in a package.
func Parse(state *core.BuildState, label, dependent core.BuildLabel) (*core.Package, error) {
	subrepo, err := checkSubrepo(state, label)
	if err != nil {
		return nil, err
	}

	if subrepo != nil {
		state = subrepo.State
	}

	state.LogParseResult(label, core.PackageParsing, "Parsing...")

	if subrepo != nil && subrepo.Target != nil {
		// We have got the definition of the subrepo, but it depends on something, make sure that has been built.
		if _, err := state.Build(subrepo.Target.Label, dependent); err != nil {
			return nil, err
		}
		if err := subrepo.State.Initialise(subrepo); err != nil {
			return nil, err
		}
	}

	// Subrepo & nothing else means we just want to ensure that subrepo is present.
	if label.Subrepo != "" && label.PackageName == "" && label.Name == "" {
		// TODO(peter): is this relevant still?
		return nil, nil
	}
	pkg, err := parsePackage(state, label, dependent, subrepo)
	if err != nil {
		return nil, err
	}
	state.LogParseResult(label, core.PackageParsed, "Parsed package")
	return pkg, nil
}

// checkSubrepo checks if the label we're parsing is within a subrepo, returning that subrepo, if present in the label.
//
// The subrepo target can be inferred from the subrepo name using convention i.e. ///foo/bar//:baz has a subrepo label
// //foo:bar. checkSubrepo parses package foo, expecting a call to `subrepo()` that registers a subrepo named foo/bar,
// so it can return it.
func checkSubrepo(state *core.BuildState, label core.BuildLabel) (*core.Subrepo, error) {
	if label.Subrepo == "" {
		return nil, nil
	}

	// Check if we already have it (we expect the higher-level driver code to have arranged for it to be parsed
	// before we get here)
	if subrepo := state.Graph.Subrepo(label.Subrepo); subrepo != nil {
		return subrepo, nil
	}
	return nil, fmt.Errorf("Subrepo %s is not defined", label.Subrepo)
}

// parsePackage parses a BUILD file and adds the package to the build graph
func parsePackage(state *core.BuildState, label, dependent core.BuildLabel, subrepo *core.Subrepo) (*core.Package, error) {
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
		if err := state.Parser.ParseReader(pkg, strings.NewReader(pkgStr), &label, &dependent); err != nil {
			return nil, fmt.Errorf("failed to parse internal package: %w", err)
		}
	} else {
		filename, dir := buildFileName(state, subrepo, fileSystem, label.PackageName)
		if filename != "" {
			pkg.Filename = filename
			if err := state.Parser.ParseFile(pkg, &label, &dependent, fileSystem, filename); err != nil {
				return nil, err
			}
		} else {
			exists := core.PathExists(dir)
			// Handle quite a few cases to provide more obvious error messages.
			if dependent != core.OriginalTarget && exists {
				return nil, fmt.Errorf("%w: %s depends on %s, but there's no %s file in %s/", ErrMissingBuildFile, dependent, label, buildFileNames(state.Config.Parse.BuildFileName), dir)
			} else if dependent != core.OriginalTarget {
				return nil, fmt.Errorf("%w: %s depends on %s, but the directory %s doesn't exist: %s", ErrMissingBuildFile, dependent, label, dir, packageName)
			} else if exists {
				return nil, fmt.Errorf("%w: Can't build %s; there's no %s file in %s/", ErrMissingBuildFile, label, buildFileNames(state.Config.Parse.BuildFileName), dir)
			}
			return nil, fmt.Errorf("%w: Can't build %s; the directory %s doesn't exist", ErrMissingBuildFile, label, dir)
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
