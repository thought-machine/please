package parse

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/thought-machine/please/rules"
	"github.com/thought-machine/please/rules/bazel"
	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/parse/asp"
)

// InitParser initialises the parser engine. This is guaranteed to be called exactly once before any calls to Parse().
func InitParser(state *core.BuildState) *core.BuildState {
	if state.Parser == nil {
		p := &aspParser{parser: newAspParser(state), init: make(chan struct{})}
		state.Parser = p
		go p.Init(state)
	}
	return state
}

// aspParser implements the core.Parser interface around our parser package.
type aspParser struct {
	parser *asp.Parser
	init   chan struct{}
	once   sync.Once
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

// NewParser creates a new parser for the state
func (p *aspParser) NewParser(state *core.BuildState) {
	state.Parser = &aspParser{parser: newAspParser(state), init: make(chan struct{})}
}

func (p *aspParser) WaitForInit() {
	<-p.init
}

// getIncludesFromConfig gets the preloaded subincludes for this state, deduplicating if there are duplicates
func getIncludesFromConfig(state *core.BuildState) []core.BuildLabel {
	done := map[core.BuildLabel]struct{}{}
	includes := make([]core.BuildLabel, 0, len(state.Config.Parse.PreloadSubincludes)+len(state.RepoConfig.Parse.PreloadSubincludes))

	is := state.Config.Parse.PreloadSubincludes
	if state.RepoConfig != nil {
		is = append(is, state.RepoConfig.Parse.PreloadSubincludes...)
	}

	for _, i := range state.Config.Parse.PreloadSubincludes {
		_, ok := done[i]
		if ok {
			continue
		}

		includes = append(includes, i)
		done[i] = struct{}{}
	}
	return includes
}

func (p *aspParser) Init(state *core.BuildState) {
	p.once.Do(func() {
		includes := getIncludesFromConfig(state)
		wg := sync.WaitGroup{}
		for _, inc := range includes {
			if inc.IsPseudoTarget() {
				log.Fatalf("Can't preload pseudotarget %v", inc)
			}
			wg.Add(1)
			// Queue them up asynchronously to feed the queues as quickly as possible
			go func(inc core.BuildLabel) {
				state.WaitForBuiltTarget(inc, core.OriginalTarget)
				wg.Done()
			}(inc)
		}

		// We must wait for all the subinclude targets to be built otherwise updating the locals might race with parsing
		// a package
		wg.Wait()

		// Preload them in order to avoid non-deterministic errors when the subincludes depend on each other
		for _, inc := range includes {
			if err := p.parser.SubincludeTarget(state, state.WaitForTargetAndEnsureDownload(inc, core.OriginalTarget)); err != nil {
				log.Fatalf("%v", err)
			}
		}
		p.parser.Finalise()
		close(p.init)
	})
}

func (p *aspParser) ParseFile(pkg *core.Package, forLabel, dependent *core.BuildLabel, forSubinclude bool, filename string) error {
	return p.parser.ParseFile(pkg, forLabel, dependent, forSubinclude, filename)
}

func (p *aspParser) ParseReader(pkg *core.Package, reader io.ReadSeeker) error {
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
