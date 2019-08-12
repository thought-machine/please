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
func InitParser(state *core.BuildState) {
	if state.Parser == nil {
		state.Parser = &aspParser{asp: newAspParser(state)}
	}
}

// An aspParser implements the core.Parser interface around our asp package.
type aspParser struct {
	asp *asp.Parser
}

// newAspParser returns a asp.Parser object with all the builtins loaded
func newAspParser(state *core.BuildState) *asp.Parser {
	p := asp.NewParser(state)
	log.Debug("Loading built-in build rules...")
	dir, _ := rules.AssetDir("")
	sort.Strings(dir)
	for _, filename := range dir {
		if strings.HasSuffix(filename, ".gob") {
			srcFile := strings.TrimSuffix(filename, ".gob")
			src, _ := rules.Asset(srcFile)
			p.MustLoadBuiltins("rules/"+srcFile, src, rules.MustAsset(filename))
		}
	}

	for _, preload := range state.Config.Parse.PreloadBuildDefs {
		log.Debug("Preloading build defs from %s...", preload)
		p.MustLoadBuiltins(preload, nil, nil)
	}

	if state.Config.Bazel.Compatibility {
		// Add a subrepo for @bazel_tools which appears to be one of their builtins.
		// Mostly we only include build defs in there.
		createBazelSubrepo(state)
	}

	log.Debug("Parser initialised")
	return p
}

func (p *aspParser) ParseFile(state *core.BuildState, pkg *core.Package, filename string) error {
	return p.asp.ParseFile(pkg, filename)
}

func (p *aspParser) ParseReader(state *core.BuildState, pkg *core.Package, reader io.ReadSeeker) error {
	_, err := p.asp.ParseReader(pkg, reader)
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

// runBuildFunction runs either the pre- or post-build function.
func (p *aspParser) runBuildFunction(tid int, state *core.BuildState, target *core.BuildTarget, callbackType string, f func() error) error {
	state.LogBuildResult(tid, target.Label, core.PackageParsing, fmt.Sprintf("Running %s-build function for %s", callbackType, target.Label))
	pkg := state.WaitForPackage(target.Label)
	changed, err := pkg.EnterBuildCallback(f)
	if err != nil {
		state.LogBuildError(tid, target.Label, core.ParseFailed, err, "Failed %s-build function for %s", callbackType, target.Label)
	} else {
		if err := rescanDeps(state, changed); err != nil {
			return err
		}
		state.LogBuildResult(tid, target.Label, core.TargetBuilding, fmt.Sprintf("Finished %s-build function for %s", callbackType, target.Label))
	}
	return err
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
	filenames, _ := bazel.AssetDir("")
	for _, filename := range filenames {
		if err := ioutil.WriteFile(path.Join(dir, strings.Replace(filename, ".build_defs", ".bzl", -1)), bazel.MustAsset(filename), 0644); err != nil {
			log.Fatalf("%s", err)
		}
	}
}
