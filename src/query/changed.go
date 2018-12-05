package query

import (
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/scm"
)

// ChangedRequest is a simple parameter object
type ChangedRequest struct {
	Since            string
	DiffSpec         string
	IncludeDependees string
}

// ChangedLabels returns all BuildLabels that have changed according to the given request.
func ChangedLabels(state *core.BuildState, request ChangedRequest) []core.BuildLabel {
	workspaceChangedFiles := changedFiles(request.Since, request.DiffSpec)
	if len(workspaceChangedFiles) == 0 {
		return []core.BuildLabel{}
	}

	var labels []core.BuildLabel
	targets := targetsForChangedFiles(state.Graph, workspaceChangedFiles, request.IncludeDependees)
	for _, t := range targets {
		if state.ShouldInclude(t) {
			labels = append(labels, t.Label)
		}
	}

	return labels
}

func changedFiles(since string, diffSpec string) []string {
	if diffSpec != "" {
		return scm.ChangesIn(diffSpec, "")
	}

	if since == "" {
		since = scm.CurrentRevIdentifier()
	}

	return scm.ChangedFiles(since, true, "")
}

func targetsForChangedFiles(graph *core.BuildGraph, files []string, includeDependees string) []*core.BuildTarget {
	addresses := make(map[*core.BuildTarget]struct{})
	for _, target := range graph.AllTargets() {
		for _, file := range files {
			if target.HasAbsoluteSource(file) {
				addresses[target] = struct{}{}
			}
		}
	}
	if includeDependees != "direct" && includeDependees != "transitive" {
		keys := make([]*core.BuildTarget, len(addresses))

		i := 0
		for k := range addresses {
			keys[i] = k
			i++
		}

		return keys
	}

	dependents := make(map[*core.BuildTarget]struct{})
	if includeDependees == "direct" {
		for target := range addresses {
			dependents[target] = struct{}{}

			for _, dep := range graph.ReverseDependencies(target) {
				dependents[dep] = struct{}{}
			}
		}
	} else {
		for target := range addresses {
			visit(dependents, target, graph.ReverseDependencies)
		}
	}
	keys := make([]*core.BuildTarget, len(dependents))

	i := 0
	for k := range dependents {
		keys[i] = k
		i++
	}

	return keys
}

func visit(dependents map[*core.BuildTarget]struct{}, target *core.BuildTarget, f func(*core.BuildTarget) []*core.BuildTarget) {
	for _, dep := range f(target) {
		if _, exists := dependents[dep]; !exists {
			dependents[dep] = struct{}{}
			visit(dependents, dep, f)
		}
	}
}
