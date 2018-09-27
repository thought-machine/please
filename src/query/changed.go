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

func ChangedTargetAddresses(graph *core.BuildGraph, request ChangedRequest) []core.BuildLabel {
	workspaceChangedFiles := changedFiles(request.Since, request.DiffSpec)
	if len(workspaceChangedFiles) == 0 {
		return []core.BuildLabel{}
	}

	return TargetAddressesForChangedFiles(graph, workspaceChangedFiles, request.IncludeDependees)
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

func TargetAddressesForChangedFiles(graph *core.BuildGraph, files []string, includeDependees string) []core.BuildLabel {
	addresses := make([]core.BuildLabel, 0)
	for _, target := range graph.AllTargets() {
		for _, source := range target.Sources {
			if source.Label() == nil {
				for path := range source.Paths(graph) {
					for file := range files {
						if path == file {
							addresses = append(addresses, target.Label)
						}
					}
				}
			}
		}
	}
	if includeDependees == "" {
		return addresses
	}

	// Find dependees
	return addresses
}