package core

import (
	"encoding/json"
)

// StampFile returns the contents of a stamp file, that is the data that would be written for
// a target that is marked stamp=True.
// This file contains information about its transitive dependencies that can be used to
// embed information into the output (for example information from labels or licences).
func StampFile(target *BuildTarget) []byte {
	info := &stampInfo{
		Targets: map[BuildLabel]targetInfo{},
	}
	populateStampInfo(target, info)
	b, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		log.Fatalf("Failed to encode stamp file: %s", err)
	}
	return b
}

func populateStampInfo(target *BuildTarget, info *stampInfo) {
	info.Targets[target.Label] = targetInfo{
		Licences: target.Licences,
		Labels:   target.Labels,
	}
	for _, dep := range target.Dependencies() {
		if _, present := info.Targets[dep.Label]; !present {
			populateStampInfo(dep, info)
		}
	}
}

type stampInfo struct {
	Targets map[BuildLabel]targetInfo `json:"targets"`
}

type targetInfo struct {
	Labels   []string `json:"labels,omitempty"`
	Licences []string `json:"licences,omitempty"`
}
