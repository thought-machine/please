package core

import (
	"fmt"
	"strings"
)

type dependencyChain []*BuildLabel
type dependencyLink struct {
	from *BuildLabel
	to   *BuildLabel
}

func (c dependencyChain) String() string {
	labels := make([]string, len(c))
	for i, l := range c {
		labels[i] = l.String()
	}
	return strings.Join(labels, "\n -> ")
}

type cycleDetector struct {
	deps  map[*BuildLabel]map[*BuildLabel]struct{}
	addQueue chan dependencyLink
	removeQueue chan dependencyLink
}

func newCycleDetector() *cycleDetector {
	c := new(cycleDetector)

	// Set this to an order of magnitude that seems sensible to avoid resizing them rapidly initially
	c.deps = make(map[*BuildLabel]map[*BuildLabel]struct{}, 1000)

	// Buffer this a little so handle spikes in targets coming in
	c.addQueue = make(chan dependencyLink)
	c.removeQueue = make(chan dependencyLink)

	return c
}

// AddDependency queues up another dependency for the cycle detector to check
func (c *cycleDetector) AddDependency(from *BuildLabel, to *BuildLabel) {
	go func() {
		c.addQueue <- dependencyLink{from: from, to: to}
	}()
}

// AddDependency queues up another dependency for the cycle detector to check
func (c *cycleDetector) RemoveDependency(from *BuildLabel, to *BuildLabel) {
	go func() {
		c.removeQueue <- dependencyLink{from: from, to: to}
	}()
}

// checkForCycle just checks to see if there's a dependency cycle. It doesn't compute the cycle to avoid excess
// allocations. buildCycle can be used to reconstruct the cycle once one has been found.
func (c *cycleDetector) checkForCycle(head, tail *BuildLabel, done map[BuildLabel]struct{}) bool {
	if _, ok := done[*tail]; ok {
		return false
	}
	done[*tail] = struct{}{}
	if tailDeps, ok := c.deps[tail]; ok {
		for dep := range tailDeps {
			if dep == head {
				// If the tail has a dependency on the head, we've found a cycle
				return true
			}

			if c.checkForCycle(head, dep, done) {
				return true
			}
		}
	}
	return false
}

// buildCycle is used to actually reconstruct the cycle after we've found one
func (c *cycleDetector) buildCycle(chain []*BuildLabel, done map[BuildLabel]struct{}) []*BuildLabel {
	tail := chain[len(chain)-1]
	head := chain[0]

	if _, ok := done[*tail]; ok {
		return nil
	}
	done[*tail] = struct{}{}
	if tailDeps, ok := c.deps[tail]; ok {
		for dep := range tailDeps {
			if dep == head {
				// If the tail has a dependency on the head, we've found a cycle
				return append(chain, dep)
			}

			if newChain := c.buildCycle(append(chain, dep), done); newChain != nil {
				return newChain
			}
		}
	}
	return nil
}

func (c *cycleDetector) addDep(link dependencyLink) error {
	if c.checkForCycle(link.from, link.to, make(map[BuildLabel]struct{})) {
		return failWithGraphCycle(c.buildCycle([]*BuildLabel{link.from, link.to}, make(map[BuildLabel]struct{})))
	}
	m := c.deps[link.from]
	if m == nil {
		m = map[*BuildLabel]struct{}{}
		c.deps[link.from] = m
	}
	c.deps[link.from][link.to] = struct{}{}
	return nil
}


func failWithGraphCycle(cycle dependencyChain) error {
	return fmt.Errorf("%s \nSorry, but you'll have to refactor your build files to avoid this cycle", cycle.String())
}

func (c *cycleDetector) run() {
	go func() {
		for {
			select {
			case dep := <-c.addQueue:
				log.Warningf("%v waiting for %v", dep.from, dep.to)
				if err := c.addDep(dep); err != nil {
					log.Fatalf("Dependency cycle found:\n %v", err)
				}
			case dep := <-c.removeQueue:
				log.Warningf("%v done for %v", dep.from, dep.to)
				delete(c.deps[dep.from], dep.to)
				if len(c.deps[dep.from]) == 0 {
					delete(c.deps, dep.from)
				}
			}
		}
	}()
}
