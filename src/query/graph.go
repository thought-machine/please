package query

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path"
	"sync"

	"build"
	"core"
)

// Graph prints a representation of the build graph as JSON.
func Graph(graph *core.BuildGraph, targets []core.BuildLabel) {
	log.Notice("Generating graph...")
	g := makeJSONGraph(graph, targets)
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

func makeJSONGraph(graph *core.BuildGraph, targets []core.BuildLabel) *JSONGraph {
	ret := JSONGraph{Packages: map[string]JSONPackage{}}
	if len(targets) == 0 {
		for pkg := range makeAllPackages(graph) {
			ret.Packages[pkg.name] = pkg
		}
	} else {
		done := map[core.BuildLabel]struct{}{}
		for _, target := range targets {
			addJSONTarget(graph, &ret, target, done)
		}
	}
	return &ret
}

// makeAllPackages constructs all the JSONPackage objects for this graph in parallel.
func makeAllPackages(graph *core.BuildGraph) <-chan JSONPackage {
	ch := make(chan JSONPackage, 100)
	go func() {
		packages := graph.PackageMap()
		var wg sync.WaitGroup
		wg.Add(len(packages))
		for _, pkg := range packages {
			go func(pkg *core.Package) {
				ch <- makeJSONPackage(graph, pkg)
				wg.Done()
			}(pkg)
		}
		wg.Wait()
		close(ch)
	}()
	return ch
}

func addJSONTarget(graph *core.BuildGraph, ret *JSONGraph, label core.BuildLabel, done map[core.BuildLabel]struct{}) {
	if _, present := done[label]; present {
		return
	}
	done[label] = struct{}{}
	if label.IsAllTargets() {
		pkg := graph.PackageOrDie(label.PackageName)
		for _, target := range pkg.Targets {
			addJSONTarget(graph, ret, target.Label, done)
		}
		return
	}
	target := graph.TargetOrDie(label)
	if _, present := ret.Packages[label.PackageName]; present {
		ret.Packages[label.PackageName].Targets[label.Name] = makeJSONTarget(graph, target)
	} else {
		ret.Packages[label.PackageName] = JSONPackage{
			Targets: map[string]JSONTarget{
				label.Name: makeJSONTarget(graph, target),
			},
		}
	}
	for _, dep := range target.Dependencies() {
		addJSONTarget(graph, ret, dep.Label, done)
	}
}

func makeJSONPackage(graph *core.BuildGraph, pkg *core.Package) JSONPackage {
	targets := map[string]JSONTarget{}
	for name, target := range pkg.Targets {
		targets[name] = makeJSONTarget(graph, target)
	}
	return JSONPackage{name: pkg.Name, Targets: targets}
}

func makeJSONTarget(graph *core.BuildGraph, target *core.BuildTarget) JSONTarget {
	t := JSONTarget{}
	for in := range core.IterSources(graph, target) {
		t.Inputs = append(t.Inputs, in.Src)
	}
	for _, out := range target.Outputs() {
		t.Outputs = append(t.Outputs, path.Join(target.Label.PackageName, out))
	}
	for _, src := range target.AllSourcePaths(graph) {
		t.Sources = append(t.Sources, src)
	}
	for _, dep := range target.Dependencies() {
		t.Deps = append(t.Deps, dep.Label.String())
	}
	for data := range core.IterRuntimeFiles(graph, target, false) {
		t.Data = append(t.Data, data.Src)
	}
	t.Labels = target.Labels
	t.Requires = target.Requires
	rawHash := append(build.RuleHash(target, true, false), core.State.Hashes.Config...)
	t.Hash = base64.RawStdEncoding.EncodeToString(rawHash)
	t.Test = target.IsTest
	t.Binary = target.IsBinary
	t.TestOnly = target.TestOnly
	return t
}
