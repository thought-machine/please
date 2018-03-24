package parse

import (
	"fmt"
	"sort"
	"strings"

	"core"
	"parse/asp"
	"parse/rules"
)

// InitParser initialises the parser engine. This is guaranteed to be called exactly once before any calls to Parse().
func InitParser(state *core.BuildState) {
	state.Parser = &aspParser{asp: newAspParser(state)}
}

// An aspParser implements the core.Parser interface around our asp package.
type aspParser struct {
	asp *asp.Parser
}

// newAspParser creates and returns a new asp.Parser.
func newAspParser(state *core.BuildState) *asp.Parser {
	p := asp.NewParser(state)
	log.Debug("Loading built-in build rules...")
	dir, _ := rules.AssetDir("")
	sort.Strings(dir)
	for _, filename := range dir {
		if strings.HasSuffix(filename, ".gob") {
			srcFile := strings.TrimSuffix(filename, ".gob")
			src, _ := rules.Asset(srcFile)
			p.MustLoadBuiltins("src/parse/rules/"+srcFile, src, rules.MustAsset(filename))
		}
	}
	for _, preload := range state.Config.Parse.PreloadBuildDefs {
		log.Debug("Preloading build defs from %s...", preload)
		p.MustLoadBuiltins(preload, nil, nil)
	}
	log.Debug("Parser initialised")
	return p
}

func (p *aspParser) ParseFile(state *core.BuildState, pkg *core.Package, filename string) error {
	return p.asp.ParseFile(pkg, filename)
}

func (p *aspParser) UndeferAnyParses(state *core.BuildState, target *core.BuildTarget) {
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
	pkg := state.Graph.Package(target.Label.PackageName)
	changed, err := pkg.EnterBuildCallback(f)
	if err != nil {
		state.LogBuildError(tid, target.Label, core.ParseFailed, err, "Failed %s-build function for %s", callbackType, target.Label)
	} else {
		rescanDeps(state, changed)
		state.LogBuildResult(tid, target.Label, core.TargetBuilding, fmt.Sprintf("Finished %s-build function for %s", callbackType, target.Label))
	}
	return err
}
