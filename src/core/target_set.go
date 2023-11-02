package core

import (
	"slices"
	"sync"
)

// A TargetSet contains a series of targets and supports efficiently checking for membership
// The zero value is not safe for use.
type TargetSet struct {
	targets  map[BuildLabel]struct{}
	packages map[packageKey]struct{}
	mutex    sync.RWMutex
}

// NewTargetSet returns a new TargetSet.
func NewTargetSet() *TargetSet {
	return &TargetSet{
		targets:  map[BuildLabel]struct{}{},
		packages: map[packageKey]struct{}{},
	}
}

// Add adds a new target to this set.
func (ts *TargetSet) Add(label BuildLabel) {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()
	if label.IsAllSubpackages() {
		panic("TargetSet doesn't support ... labels")
	} else if label.IsAllTargets() {
		ts.packages[label.packageKey()] = struct{}{}
	} else {
		ts.targets[label] = struct{}{}
	}
}

// Match checks if this label is covered by the set (either explicitly or via :all)
func (ts *TargetSet) Match(label BuildLabel) bool {
	ts.mutex.RLock()
	defer ts.mutex.RUnlock()
	if _, present := ts.targets[label]; present {
		return true
	}
	_, present := ts.packages[label.packageKey()]
	return present
}

// MatchExact checks if this label was explicitly added to the set (i.e. :all doesn't count)
func (ts *TargetSet) MatchExact(label BuildLabel) bool {
	ts.mutex.RLock()
	defer ts.mutex.RUnlock()
	_, present := ts.targets[label]
	return present
}

// AllTargets returns a copy of the set of targets
func (ts *TargetSet) AllTargets() []BuildLabel {
	ts.mutex.RLock()
	defer ts.mutex.RUnlock()
	ret := make([]BuildLabel, 0, len(ts.targets)+len(ts.packages))
	for target := range ts.targets {
		ret = append(ret, target)
	}
	for pkg := range ts.packages {
		ret = append(ret, pkg.BuildLabel())
	}
	slices.SortFunc(ret, BuildLabel.Compare)
	return ret
}
