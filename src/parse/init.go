package parse

import (
	"context"
	"fmt"
	"io"
	iofs "io/fs"
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

// InitParser initialises the parser engine.
func InitParser(state *core.BuildState, callbacks asp.Callbacks) *asp.Parser {
	// There is some awkward coupling here for the benefit of the language server, which wants to get its
	// hands on the parser, but it cannot create a fully functional one any more.
	if p, ok := state.Parser.(*aspParser); ok {
		p.callbacks = callbacks
		p.parser.SetCallbacks(callbacks)
		return p.parser
	}
	p := newAspParser(state, callbacks)
	state.Parser = &aspParser{parser: p, callbacks: callbacks}
	return p
}

// aspParser implements the core.Parser interface around our parser package.
type aspParser struct {
	parser    *asp.Parser
	callbacks asp.Callbacks
}

// newAspParser returns a asp.Parser object with all the builtins loaded
func newAspParser(state *core.BuildState, callbacks asp.Callbacks) *asp.Parser {
	p := asp.NewParser(state, callbacks)
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

func (p *aspParser) ParseFile(ctx context.Context, pkg *core.Package, forLabel, dependent *core.BuildLabel, fs iofs.FS, filename string) error {
	return p.parser.ParseFile(ctx, pkg, forLabel, dependent, fs, filename)
}

func (p *aspParser) ParseReader(ctx context.Context, pkg *core.Package, reader io.ReadSeeker, forLabel, dependent *core.BuildLabel) error {
	_, err := p.parser.ParseReader(ctx, pkg, reader, forLabel, dependent)
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

// runBuildFunction runs either the pre- or post-build function.
func (p *aspParser) runBuildFunction(state *core.BuildState, target *core.BuildTarget, callbackType string, f func() error) error {
	state.LogBuildResult(target, core.PackageParsing, fmt.Sprintf("Running %s-build function for %s", callbackType, target.Label))
	// This doesn't re-parse anything; it waits for the parse of this target's package to complete if
	// one is still in flight. Targets are added to the graph as each rule is created, so a target can
	// be picked up and built before the file that defines it has been fully interpreted - but the
	// callback both reads the package out of the graph and mutates it, so it can't run until then.
	// There's no parse in flight for us to inherit a context from, hence Background.
	if _, err := p.callbacks.Parse(context.Background(), target.Label, target.Label); err != nil {
		return err
	}
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

// BuildRuleArgOrder returns a map of the arguments to build rule and the order they appear in the source file
func BuildRuleArgOrder(state *core.BuildState) map[string]int {
	p := asp.NewParser(state, nil)
	b, _ := rules.ReadAsset("builtins.build_defs")
	stmts, _ := p.ParseData(b, "builtins.build_defs")
	m := map[string]int{}
	for _, stmt := range stmts {
		if stmt.FuncDef != nil && stmt.FuncDef.Name == "build_rule" {
			for i, arg := range stmt.FuncDef.Arguments {
				m[arg.Name] = i
			}
			return m
		}
	}
	return m
}
