package query

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path"
	"sync"

	"github.com/thought-machine/please/src/build"
	"github.com/thought-machine/please/src/core"
)

// Graph prints a representation of the build graph as JSON.
func Graph(state *core.BuildState, targets []core.BuildLabel) {
	log.Notice("Generating graph...")
	g := makeJSONGraph(state, targets)
	log.Notice("Marshalling...")
	b, err := json.MarshalIndent(g, "", "    ")
	if err != nil {
		log.Fatalf("Failed to serialise JSON: %s\n", err)
	}
	log.Notice("Writing...")
	fmt.Println(string(b))
	log.Notice("Done")
}

// JSONGraph is an alternate representation of our build graph; will contain different information
// to the structures in core (also those ones can't be printed as JSON directly).
type JSONGraph struct {
	Packages map[string]JSONPackage `json:"packages"`
}

// JSONPackage is an alternate representation of a build package
type JSONPackage struct {
	name    string
	Targets map[string]JSONTarget `json:"targets"`
}

// JSONTarget is an alternate representation of a build target
type JSONTarget struct {
	Inputs   []string `json:"inputs,omitempty" note:"declared inputs of target"`
	Outputs  []string `json:"outs,omitempty" note:"corresponds to outs in rule declaration"`
	Sources  []string `json:"srcs,omitempty" note:"corresponds to srcs in rule declaration"`
	Deps     []string `json:"deps,omitempty" note:"corresponds to deps in rule declaration"`
	Data     []string `json:"data,omitempty" note:"corresponds to data in rule declaration"`
	Labels   []string `json:"labels,omitempty" note:"corresponds to labels in rule declaration"`
	Requires []string `json:"requires,omitempty" note:"corresponds to requires in rule declaration"`
	Hash     string   `json:"hash" note:"partial hash of target, does not include source hash"`
	Test     bool     `json:"test,omitempty" note:"true if target is a test"`
	Binary   bool     `json:"binary,omitempty" note:"true if target is a binary"`
	TestOnly bool     `json:"test_only,omitempty" note:"true if target should be restricted to test code"`
}

func makeJSONGraph(state *core.BuildState, targets []core.BuildLabel) *JSONGraph {
	ret := JSONGraph{Packages: map[string]JSONPackage{}}
	if len(targets) == 0 {
		for pkg := range makeAllPackages(state) {
			ret.Packages[pkg.name] = pkg
		}
	} else {
		done := map[core.BuildLabel]struct{}{}
		for _, target := range targets {
			addJSONTarget(state, &ret, target, done)
		}
	}
	return &ret
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

func addJSONTarget(state *core.BuildState, ret *JSONGraph, label core.BuildLabel, done map[core.BuildLabel]struct{}) {
	if _, present := done[label]; present {
		return
	}
	done[label] = struct{}{}
	if label.IsAllTargets() {
		pkg := state.Graph.PackageOrDie(label)
		for _, target := range pkg.AllTargets() {
			addJSONTarget(state, ret, target.Label, done)
		}
		return
	}
	target := state.Graph.TargetOrDie(label)
	if _, present := ret.Packages[label.PackageName]; present {
		ret.Packages[label.PackageName].Targets[label.Name] = makeJSONTarget(state, target)
	} else {
		ret.Packages[label.PackageName] = JSONPackage{
			Targets: map[string]JSONTarget{
				label.Name: makeJSONTarget(state, target),
			},
		}
	}
	for _, dep := range target.Dependencies() {
		addJSONTarget(state, ret, dep.Label, done)
	}
}

func makeJSONPackage(state *core.BuildState, pkg *core.Package) JSONPackage {
	targets := map[string]JSONTarget{}
	for _, target := range pkg.AllTargets() {
		targets[target.Label.Name] = makeJSONTarget(state, target)
	}
	return JSONPackage{name: pkg.Name, Targets: targets}
}

func makeJSONTarget(state *core.BuildState, target *core.BuildTarget) JSONTarget {
	t := JSONTarget{
		Sources: target.AllSourcePaths(state.Graph),
	}
	for in := range core.IterSources(state.Graph, target) {
		t.Inputs = append(t.Inputs, in.Src)
	}
	for _, out := range target.Outputs() {
		t.Outputs = append(t.Outputs, path.Join(target.Label.PackageName, out))
	}
	for _, dep := range target.Dependencies() {
		t.Deps = append(t.Deps, dep.Label.String())
	}
	for data := range core.IterRuntimeFiles(state.Graph, target, false) {
		t.Data = append(t.Data, data.Src)
	}
	t.Labels = target.Labels
	t.Requires = target.Requires
	rawHash := append(build.RuleHash(state, target, true, false), state.Hashes.Config...)
	t.Hash = base64.RawStdEncoding.EncodeToString(rawHash)
	t.Test = target.IsTest
	t.Binary = target.IsBinary
	t.TestOnly = target.TestOnly
	return t
}
