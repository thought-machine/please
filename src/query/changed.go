package query

import (
	"core"
	"scm"
)

type ChangedRequest struct {
	Since            string
	DiffSpec         string
	IncludeDependees string
}

func ChangedLabels(state *core.BuildState, request ChangedRequest) []core.BuildLabel {
	workspaceChangedFiles := changedFiles(request.Since, request.DiffSpec)
	if len(workspaceChangedFiles) == 0 {
		return []core.BuildLabel{}
	}

	labels := make([]core.BuildLabel, 0)
	targets := TargetsForChangedFiles(state.Graph, workspaceChangedFiles, request.IncludeDependees)
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

func TargetsForChangedFiles(graph *core.BuildGraph, files []string, includeDependees string) []*core.BuildTarget {
	addresses := make([]*core.BuildTarget, 0)
	for _, target := range graph.AllTargets() {
		for _, source := range target.Sources {
			if source.Label() == nil {
				for _, path := range source.Paths(graph) {
					for _, file := range files {
						if path == file {
							addresses = append(addresses, target)
						}
					}
				}
			}
		}
	}
	if includeDependees != "direct" && includeDependees != "transitive" {
		return addresses
	}

	dependents := make(map[*core.BuildTarget]struct{})
	if includeDependees == "direct" {
		for _, target := range addresses {
			dependents[target] = struct{}{}

			for _, dep := range graph.ReverseDependencies(target) {
				dependents[dep] = struct{}{}
			}
		}
	} else {
		for _, target := range addresses {
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
		dependents[dep] = struct{}{}
		visit(dependents, dep, f)
	}
}
