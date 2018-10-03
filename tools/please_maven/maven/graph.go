package maven

import (
	"strings"
)

// A Graph is a minimal representation of the parts of `plz query graph`'s output that we care about.
type Graph struct {
	Packages       map[string]pkg `json:"packages"`
	mavenToPackage map[string]string
}

type pkg struct {
	Targets map[string]target `json:"targets"`
}

type target struct {
	Labels []string `json:"labels,omitempty"`
}

// BuildMapping sets up the internal reverse mapping of maven id -> target.
// It must be called once before anything else is.
func (g *Graph) BuildMapping() {
	g.mavenToPackage = map[string]string{}
	for pkgName, pkg := range g.Packages {
		for targetName, target := range pkg.Targets {
			for _, label := range target.Labels {
				if parts := strings.Split(label, ":"); len(parts) > 3 && parts[0] == "mvn" {
					g.mavenToPackage[parts[1]+":"+parts[2]] = "//" + pkgName + ":" + targetName
				}
			}
		}
	}
}

// Needed returns true if we need a build rule for the given group ID / artifact ID.
// It's false if one already exists in the current build files.
func (g *Graph) Needed(groupID, artifactID string) bool {
	if _, present := g.mavenToPackage[groupID+":"+artifactID]; present {
		log.Debug("Dependency %s:%s not needed, already exists in graph", groupID, artifactID)
		return false
	}
	return true
}

// Dep returns the dependency for a given groupID / artifact ID.
// If it's not in the graph already it returns the label for a newly added target.
func (g *Graph) Dep(groupID, artifactID string) string {
	if dep, present := g.mavenToPackage[groupID+":"+artifactID]; present {
		return dep
	}
	return ":" + artifactID
}
