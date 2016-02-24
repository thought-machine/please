package query

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path"

	"build"
	"core"
)

// QueryGraph prints a representation of the build graph as JSON.
func QueryGraph(graph *core.BuildGraph, targets []core.BuildLabel) {
	b, err := json.MarshalIndent(makeJSONGraph(graph, targets), "", "    ")
	if err != nil {
		log.Fatalf("Failed to serialise JSON: %s\n", err)
	}
	fmt.Println(string(b))
}

// JSONGraph is an alternate representation of our build graph; will contain different information
// to the structures in core (also those ones can't be printed as JSON directly).
type JSONGraph struct {
	Packages map[string]JSONPackage `json:"packages"`
}

// JSONPackage is an alternate representation of a build package
type JSONPackage struct {
	Targets map[string]JSONTarget `json:"targets"`
}

// JSONTarget is an alternate representation of a build target
type JSONTarget struct {
	Inputs  []string `json:"inputs,omitempty" note:"declared inputs of target"`
	Outputs []string `json:"outs,omitempty" note:"corresponds to outs in rule declaration"`
	Sources []string `json:"srcs,omitempty" note:"corresponds to srcs in rule declaration"`
	Deps    []string `json:"deps,omitempty" note:"corresponds to deps in rule declaration"`
	Labels  []string `json:"labels,omitempty" note:"corresponds to labels in rule declaration"`
	Hash    string   `json:"hash" note:"partial hash of target, does not include source hash"`
	Test    bool     `json:"test,omitempty" note:"true if target is a test"`
}

func makeJSONGraph(graph *core.BuildGraph, targets []core.BuildLabel) *JSONGraph {
	ret := JSONGraph{Packages: map[string]JSONPackage{}}
	if len(targets) == 0 {
		for name, pkg := range graph.PackageMap() {
			ret.Packages[name] = makeJSONPackage(graph, pkg)
		}
	} else {
		done := map[core.BuildLabel]struct{}{}
		for _, target := range targets {
			addJSONTarget(graph, &ret, target, done)
		}
	}
	return &ret
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
	for _, dep := range target.Dependencies {
		addJSONTarget(graph, ret, dep.Label, done)
	}
}

func makeJSONPackage(graph *core.BuildGraph, pkg *core.Package) JSONPackage {
	targets := map[string]JSONTarget{}
	for name, target := range pkg.Targets {
		targets[name] = makeJSONTarget(graph, target)
	}
	return JSONPackage{Targets: targets}
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
	for _, dep := range target.DeclaredDependencies {
		t.Deps = append(t.Deps, dep.String())
	}
	t.Labels = target.Labels
	rawHash := append(build.RuleHash(target, true, false), core.State.Hashes.Config...)
	t.Hash = base64.RawStdEncoding.EncodeToString(rawHash)
	t.Test = target.IsTest
	return t
}
