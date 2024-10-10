package query

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/thought-machine/please/src/build"
	"github.com/thought-machine/please/src/core"
)

// Graph prints a representation of the build graph as JSON.
func Graph(state *core.BuildState, targets []core.BuildLabel) {
	log.Notice("Generating graph...")
	g := makeJSONGraph(state, targets)

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "    ")
	encoder.SetEscapeHTML(false)

	log.Notice("Encoding...")
	if err := encoder.Encode(g); err != nil {
		log.Fatalf("Failed to serialise JSON: %s\n", err)
	}
	log.Notice("Done")
}

// JSONGraph is an alternate representation of our build graph; will contain different information
// to the structures in core (also those ones can't be printed as JSON directly).
type JSONGraph struct {
	Packages map[string]JSONPackage `json:"packages"`
	Subrepos map[string]*JSONGraph  `json:"subrepos,omitempty"`
}

// JSONPackage is an alternate representation of a build package
type JSONPackage struct {
	name    string
	subrepo string
	Targets map[string]JSONTarget `json:"targets"`
}

// JSONTarget is an alternate representation of a build target
type JSONTarget struct {
	Inputs   []string    `json:"inputs,omitempty" note:"declared inputs of target"`
	Outputs  []string    `json:"outs,omitempty" note:"corresponds to outs in rule declaration"`
	Sources  interface{} `json:"srcs,omitempty" note:"corresponds to srcs in rule declaration"`
	Tools    interface{} `json:"tools,omitempty" note:"corresponds to tools in rule declaration"`
	Deps     []string    `json:"deps,omitempty" note:"corresponds to deps in rule declaration"`
	Data     []string    `json:"data,omitempty" note:"corresponds to data in rule declaration"`
	Labels   []string    `json:"labels,omitempty" note:"corresponds to labels in rule declaration"`
	Requires []string    `json:"requires,omitempty" note:"corresponds to requires in rule declaration"`
	Command  string      `json:"command,omitempty" note:"the currently active command of the target. not present on filegroup or remote_file actions"`
	Hash     string      `json:"hash" note:"partial hash of target, does not include source hash"`
	Test     bool        `json:"test,omitempty" note:"true if target is a test"`
	Binary   bool        `json:"binary,omitempty" note:"true if target is a binary"`
	TestOnly bool        `json:"test_only,omitempty" note:"true if target should be restricted to test code"`
}

func makeJSONGraph(state *core.BuildState, targets []core.BuildLabel) *JSONGraph {
	ret := JSONGraph{
		Packages: map[string]JSONPackage{},
		Subrepos: map[string]*JSONGraph{},
	}
	if len(targets) == 0 {
		for pkg := range makeAllPackages(state) {
			ret.Subrepo(pkg.subrepo).Packages[pkg.name] = pkg
		}
	} else {
		done := map[core.BuildLabel]struct{}{}
		for _, target := range targets {
			addJSONTarget(state, &ret, target, done)
		}
	}
	return &ret
}

// Subrepo returns a subrepo for the given name. If it's empty the top-level repo is returned.
func (graph *JSONGraph) Subrepo(name string) *JSONGraph {
	if name == "" {
		return graph
	} else if subrepo, present := graph.Subrepos[name]; present {
		return subrepo
	}
	subrepo := &JSONGraph{
		Packages: map[string]JSONPackage{},
		Subrepos: map[string]*JSONGraph{},
	}
	graph.Subrepos[name] = subrepo
	return subrepo
}

// makeAllPackages constructs all the JSONPackage objects for this graph in parallel.
func makeAllPackages(state *core.BuildState) <-chan JSONPackage {
	ch := make(chan JSONPackage, 100)
	go func() {
		packages := state.Graph.PackageMap()
		var wg sync.WaitGroup
		wg.Add(len(packages))
		for _, pkg := range packages {
			go func(pkg *core.Package) {
				ch <- makeJSONPackage(state, pkg)
				wg.Done()
			}(pkg)
		}
		wg.Wait()
		close(ch)
	}()
	return ch
}

func addJSONTarget(state *core.BuildState, graph *JSONGraph, label core.BuildLabel, done map[core.BuildLabel]struct{}) {
	if _, present := done[label]; present {
		return
	}
	done[label] = struct{}{}
	if label.IsAllTargets() {
		pkg := state.Graph.PackageOrDie(label)
		for _, target := range pkg.AllTargets() {
			addJSONTarget(state, graph, target.Label, done)
		}
		return
	}
	target := state.Graph.TargetOrDie(label)
	repo := graph.Subrepo(label.Subrepo)
	if _, present := repo.Packages[label.PackageName]; present {
		repo.Packages[label.PackageName].Targets[label.Name] = makeJSONTarget(state, target)
	} else {
		repo.Packages[label.PackageName] = JSONPackage{
			Targets: map[string]JSONTarget{
				label.Name: makeJSONTarget(state, target),
			},
		}
	}
	for _, dep := range target.Dependencies() {
		addJSONTarget(state, graph, dep.Label, done)
	}
}

func makeJSONPackage(state *core.BuildState, pkg *core.Package) JSONPackage {
	targets := map[string]JSONTarget{}
	for _, target := range pkg.AllTargets() {
		targets[target.Label.Name] = makeJSONTarget(state, target)
	}
	return JSONPackage{name: pkg.Name, subrepo: pkg.SubrepoName, Targets: targets}
}

func makeJSONTarget(state *core.BuildState, target *core.BuildTarget) JSONTarget {
	t := JSONTarget{
		Sources: makeJSONInputField(state.Graph, target.AllSourcePaths(state.Graph), target.NamedSources),
		Tools:   makeJSONInputField(state.Graph, buildInputsToStrings(state.Graph, target.AllTools()), target.AllNamedTools()),
	}
	for in := range core.IterSources(state, state.Graph, target, false) {
		t.Inputs = append(t.Inputs, in)
	}
	for _, out := range target.Outputs() {
		t.Outputs = append(t.Outputs, filepath.Join(target.Label.PackageName, out))
	}
	for _, dep := range target.Dependencies() {
		t.Deps = append(t.Deps, dep.Label.String())
	}
	// just use run 1 as this is only used to print the test dir
	for data := range core.IterRuntimeFiles(state.Graph, target, false, target.TestDir(1)) {
		t.Data = append(t.Data, data)
	}
	t.Labels = target.Labels
	t.Requires = target.Requires
	if !target.IsFilegroup && !target.IsRemoteFile {
		t.Command = target.GetCommand(state)
	}
	rawHash := append(build.RuleHash(state, target, true, false), state.Hashes.Config...)
	t.Hash = base64.RawStdEncoding.EncodeToString(rawHash)
	t.Test = target.IsTest()
	t.Binary = target.IsBinary
	t.TestOnly = target.TestOnly
	return t
}

// makeJSONInputField takes a named and unnamed field (e.g. srcs or tools) and returns an
// appropriate representation.
func makeJSONInputField(graph *core.BuildGraph, unnamed []string, named map[string][]core.BuildInput) interface{} {
	if len(named) == 0 {
		if len(unnamed) == 0 {
			return nil
		}
		return unnamed
	}
	namedInputs := make(map[string][]string, len(named))
	for name, srcs := range named {
		namedInputs[name] = buildInputsToStrings(graph, srcs)
	}
	return namedInputs
}

func buildInputsToStrings(graph *core.BuildGraph, inputs []core.BuildInput) []string {
	s := make([]string, 0, len(inputs))
	for _, x := range inputs {
		s = append(s, x.Paths(graph)...)
	}
	return s
}
