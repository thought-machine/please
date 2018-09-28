package query

import (
	"core"
	"scm"
)

type ChangedRequest struct {
	Since string
	DiffSpec string
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
		if state.ShouldInclude(&t) {
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

func TargetsForChangedFiles(graph *core.BuildGraph, files []string, includeDependees string) []core.BuildTarget {
	addresses := make([]core.BuildTarget, 0)
	for _, target := range graph.AllTargets() {
		for _, source := range target.Sources {
			if source.Label() == nil {
				for _, path := range source.Paths(graph) {
					for _, file := range files {
						if path == file {
							addresses = append(addresses, *target)
						}
					}
				}
			}
		}
	}
	if includeDependees == "" {
		return addresses
	}

	// Find dependees - either `direct` or `transitive`
	return addresses
}