// Package misc contains utility functions, mostly to help the graph differ.
package misc

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sort"

	"gopkg.in/op/go-logging.v1"

	"core"
	"query"
)

var log = logging.MustGetLogger("misc")

// ParseGraphOrDie reads a graph file, or dies if anything goes wrong.
func ParseGraphOrDie(filename string) *query.JSONGraph {
	graph := query.JSONGraph{}
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("Error reading graph: %s", err)
	}
	if err := json.Unmarshal(data, &graph); err != nil {
		log.Fatalf("Error parsing graph: %s", err)
	}
	return &graph
}

// DiffGraphs calculates the differences between two graphs.
func DiffGraphs(before, after *query.JSONGraph, changedFiles, include, exclude []string, recurse bool) []core.BuildLabel {
	changedFileMap := toMap(changedFiles)
	allChanges := map[string]bool{}
	for pkgName, afterPkg := range after.Packages {
		beforePkg, present := before.Packages[pkgName]
		if !present {
			// Package didn't exist before, add every target in it.
			for targetName := range afterPkg.Targets {
				label := core.BuildLabel{PackageName: pkgName, Name: targetName}
				allChanges[label.String()] = true
			}
			continue
		}
		for targetName, afterTarget := range afterPkg.Targets {
			beforeTarget := beforePkg.Targets[targetName]
			if targetChanged(&beforeTarget, &afterTarget, pkgName, changedFileMap) {
				label := core.BuildLabel{PackageName: pkgName, Name: targetName}
				allChanges[label.String()] = true
			}
		}
	}
	// Now we have all the targets that are directly changed, we locate all transitive ones
	// in a second pass. We can't do this above because we've got no sensible ordering for it.
	ret := core.BuildLabels{}
	for pkgName, pkg := range after.Packages {
		for targetName, target := range pkg.Targets {
			if depsChanged(after, allChanges, pkgName, targetName, recurse) &&
				shouldInclude(&target, include, exclude) {
				ret = append(ret, core.BuildLabel{PackageName: pkgName, Name: targetName})
			}
		}
	}
	sort.Sort(ret)
	return ret
}

func targetChanged(before, after *query.JSONTarget, pkgName string, changedFiles map[string]bool) bool {
	if before.Hash != after.Hash {
		return true
	}
	// Note that if the set of sources etc has changed, the hash will have changed also,
	// so here we're only worrying about the content.
	for _, src := range after.Sources {
		if _, present := changedFiles[src]; present {
			return true
		}
	}
	// Same for data files.
	for _, data := range after.Data {
		if _, present := changedFiles[data]; present {
			return true
		}
	}
	return false
}

// depsChanged returns true if any of the transitive dependencies of this target have changed.
// It marks any changes in allChanges for efficiency.
func depsChanged(graph *query.JSONGraph, allChanges map[string]bool, pkgName, targetName string, recurse bool) bool {
	label := fmt.Sprintf("//%s:%s", pkgName, targetName)
	changed, present := allChanges[label]
	if present {
		return changed
	}
	target := graph.Packages[pkgName].Targets[targetName]
	if !recurse {
		return false
	}
	for _, dep := range target.Deps {
		depLabel := core.ParseBuildLabel(dep, "")
		if depsChanged(graph, allChanges, depLabel.PackageName, depLabel.Name, recurse) {
			allChanges[label] = true
			return true
		}
	}
	allChanges[label] = false
	return false
}

// toMap is a utility function to convert a slice of strings to a map for faster lookup.
func toMap(in []string) map[string]bool {
	ret := map[string]bool{}
	for _, s := range in {
		ret[s] = true
	}
	return ret
}

// shouldInclude returns true if the given combination of labels means we should return this target.
func shouldInclude(target *query.JSONTarget, include, exclude []string) bool {
	return (len(include) == 0 || hasAnyLabel(target, include)) && !hasAnyLabel(target, exclude)
}

func hasAnyLabel(target *query.JSONTarget, labels []string) bool {
	for _, l1 := range labels {
		for _, l2 := range target.Labels {
			if l1 == l2 {
				return true
			}
		}
	}
	return false
}
