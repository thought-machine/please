package parse

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
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
		p := &aspParser{parser: newAspParser(state)}
		state.Parser = p
	}
	return state
}

// aspParser implements the core.Parser interface around our parser package.
type aspParser struct {
	parser *asp.Parser
}

// newAspParser returns a asp.Parser object with all the builtins loaded
func newAspParser(state *core.BuildState) *asp.Parser {
	p := asp.NewParser(state)
	log.Debug("Loading built-in build rules...")
	dir, _ := rules.AllAssets()
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

func (p *aspParser) ParseFile(pkg *core.Package, forLabel, dependent *core.BuildLabel, mode core.ParseMode, filename string) error {
	return p.parser.ParseFile(pkg, forLabel, dependent, mode, filename)
}

func (p *aspParser) ParseReader(pkg *core.Package, reader io.ReadSeeker, forLabel, dependent *core.BuildLabel, mode core.ParseMode) error {
	_, err := p.parser.ParseReader(pkg, reader, forLabel, dependent, mode)
	return err
}

func (p *aspParser) RunPreBuildFunction(state *core.BuildState, target *core.BuildTarget) error {
	return p.runBuildFunction(state, target, "pre", func() error {
		return target.PreBuildFunction.Call(target)
	})
}

func (p *aspParser) RunPostBuildFunction(state *core.BuildState, target *core.BuildTarget, output string) error {
	return p.runBuildFunction(state, target, "post", func() error {
		log.Debug("Running post-build function for %s. Build output:\n%s", target.Label, output)
		return target.PostBuildFunction.Call(target, output)
	})
}

// BuildRuleArgOrder returns a map of the arguments to build rule and the order they appear in the source file
func (p *aspParser) BuildRuleArgOrder() map[string]int {
	return p.parser.BuildRuleArgOrder()
}

// RegisterPreload pre-registers a preload, forcing us to build any transitive preloads before we move on
func (p *aspParser) RegisterPreload(label core.BuildLabel) error {
	return p.parser.RegisterPreload(label)
}

// runBuildFunction runs either the pre- or post-build function.
func (p *aspParser) runBuildFunction(state *core.BuildState, target *core.BuildTarget, callbackType string, f func() error) error {
	state.LogBuildResult(target, core.PackageParsing, fmt.Sprintf("Running %s-build function for %s", callbackType, target.Label))
	state.SyncParsePackage(target.Label)
	if err := f(); err != nil {
		state.LogBuildError(target.Label, core.ParseFailed, err, "Failed %s-build function for %s", callbackType, target.Label)
		return err
	}
	state.LogBuildResult(target, core.TargetBuilding, fmt.Sprintf("Finished %s-build function for %s", callbackType, target.Label))
	return nil
}

func createBazelSubrepo(state *core.BuildState) {
	if sr := state.Graph.Subrepo("bazel_tools"); sr != nil {
		return
	}
	dir := filepath.Join(core.OutDir, "bazel_tools")
	state.Graph.AddSubrepo(core.NewSubrepo(state, "bazel_tools", dir, nil, cli.HostArch(), false))
	// TODO(peterebden): This is a bit yuck... would be nice if we could avoid hardcoding all
	//                   this upfront and add a build target to do it for us.
	dir = filepath.Join(dir, "tools/build_defs/repo")
	if err := os.MkdirAll(dir, core.DirPermissions); err != nil {
		log.Fatalf("%s", err)
	}
	for filename, data := range bazel.AllFiles() {
		if err := os.WriteFile(filepath.Join(dir, strings.ReplaceAll(filename, ".build_defs", ".bzl")), data, 0644); err != nil {
			log.Fatalf("%s", err)
		}
	}
}
