package parse

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/thought-machine/please/rules"
	"github.com/thought-machine/please/rules/bazel"
	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/parse/asp"
)

// InitParser initialises the parser engine. This is guaranteed to be called exactly once before any calls to Parse().
func InitParser(state *core.BuildState) *core.BuildState {
	if state.Parser == nil {
		state.Parser = &aspParser{parser: newAspParser(state)}
	}
	return state
}

// aspParser implements the core.Parser interface around our parser package.
type aspParser struct {
	parser *asp.Parser
}

func buildPreamble(state *core.BuildState, pkg *core.Package) string {
	if pkg.Subrepo != nil {
		// TODO(jpoole): remove this once #1436 has been addressed
		// Please doesn't respect the subrepo .plzconfig so subincludes will be added
		// in the subrepo erroneously if we don't return early here
		return ""
	}

	subincludes := make([]string, 0, len(state.Config.Parse.PreloadSubincludes))
	for _, inc := range state.Config.Parse.PreloadSubincludes {
		l := core.ParseBuildLabel(inc, pkg.Name)
		// If pkg is the package we're pre-loading subincludes from, or if it contains it's subrepo, skip.
		// N.B. we can't cross subincludes so we have to exclude all subincludes, not just the one defined in this
		// package
		pkgName := l.SubrepoLabel().PackageName
		if pkgName == "" {
			pkgName = l.PackageName
		}

		if pkg.Name == pkgName {
			return ""
		}

		subincludes = append(subincludes, fmt.Sprintf("\"%s\"", inc))
	}

	if len(subincludes) > 0 {
		return fmt.Sprintf("subinclude(%s)", strings.Join(subincludes, ", "))
	}
	return ""
}

// newAspParser returns a asp.Parser object with all the builtins loaded
func newAspParser(state *core.BuildState) *asp.Parser {
	p := asp.NewParser(state)
	log.Debug("Loading built-in build rules...")
	dir, _ := rules.AllAssets(state.ExcludedBuiltinRules())
	sort.Strings(dir)
	for _, filename := range dir {
		src, _ := rules.ReadAsset(filename)
		p.MustLoadBuiltins(filename, src)
	}

	for _, preload := range state.Config.Parse.PreloadBuildDefs {
		log.Debug("Preloading build defs from %s...", preload)
		p.MustLoadBuiltins(preload, nil)
	}

	if state.Config.Bazel.Compatibility {
		// Add a subrepo for @bazel_tools which appears to be one of their builtins.
		// Mostly we only include build defs in there.
		createBazelSubrepo(state)
	}

	log.Debug("parser initialised")
	return p
}

func (p *aspParser) ParseFile(state *core.BuildState, pkg *core.Package, filename string) error {
	if pkg.Name == "" {
		return p.parser.ParseFile(pkg, filename, "")
	}

	return p.parser.ParseFile(pkg, filename, buildPreamble(state, pkg))
}

func (p *aspParser) ParseReader(state *core.BuildState, pkg *core.Package, reader io.ReadSeeker) error {
	_, err := p.parser.ParseReader(pkg, reader)
	return err
}

func (p *aspParser) RunPreBuildFunction(threadID int, state *core.BuildState, target *core.BuildTarget) error {
	return p.runBuildFunction(threadID, state, target, "pre", func() error {
		return target.PreBuildFunction.Call(target)
	})
}

func (p *aspParser) RunPostBuildFunction(threadID int, state *core.BuildState, target *core.BuildTarget, output string) error {
	return p.runBuildFunction(threadID, state, target, "post", func() error {
		log.Debug("Running post-build function for %s. Build output:\n%s", target.Label, output)
		return target.PostBuildFunction.Call(target, output)
	})
}

// BuildRuleArgOrder returns a map of the arguments to build rule and the order they appear in the source file
func (p *aspParser) BuildRuleArgOrder() map[string]int {
	return p.parser.BuildRuleArgOrder()
}

// runBuildFunction runs either the pre- or post-build function.
func (p *aspParser) runBuildFunction(tid int, state *core.BuildState, target *core.BuildTarget, callbackType string, f func() error) error {
	state.LogBuildResult(tid, target, core.PackageParsing, fmt.Sprintf("Running %s-build function for %s", callbackType, target.Label))
	state.SyncParsePackage(target.Label)
	if err := f(); err != nil {
		state.LogBuildError(tid, target.Label, core.ParseFailed, err, "Failed %s-build function for %s", callbackType, target.Label)
		return err
	}
	state.LogBuildResult(tid, target, core.TargetBuilding, fmt.Sprintf("Finished %s-build function for %s", callbackType, target.Label))
	return nil
}

func createBazelSubrepo(state *core.BuildState) {
	dir := path.Join(core.OutDir, "bazel_tools")
	state.Graph.AddSubrepo(&core.Subrepo{
		Name:  "bazel_tools",
		Root:  dir,
		State: state,
		Arch:  cli.HostArch(),
	})
	// TODO(peterebden): This is a bit yuck... would be nice if we could avoid hardcoding all
	//                   this upfront and add a build target to do it for us.
	dir = path.Join(dir, "tools/build_defs/repo")
	if err := os.MkdirAll(dir, core.DirPermissions); err != nil {
		log.Fatalf("%s", err)
	}
	for filename, data := range bazel.AllFiles() {
		if err := ioutil.WriteFile(path.Join(dir, strings.ReplaceAll(filename, ".build_defs", ".bzl")), data, 0644); err != nil {
			log.Fatalf("%s", err)
		}
	}
}
