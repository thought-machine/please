package core

import (
	"fmt"
	"strings"
)

type dependencyChain []BuildLabel
type dependencyLink struct {
	from BuildLabel
	to   BuildLabel
}

func (c dependencyChain) String() string {
	labels := make([]string, len(c))
	for i, l := range c {
		labels[i] = l.String()
	}
	return strings.Join(labels, "\n -> ")
}

type cycleDetector struct {
	deps  map[BuildLabel][]BuildLabel
	queue chan dependencyLink
}

func newCycleDetector() *cycleDetector {
	c := new(cycleDetector)

	// Set these
	c.deps = make(map[BuildLabel][]BuildLabel, 1000)
	c.queue = make(chan dependencyLink, 1000)

	return c
}

// AddDependency queues up another dependency for the cycle detector to check
func (c *cycleDetector) AddDependency(depending BuildLabel, dep BuildLabel) {
	go func() {
		c.queue <- dependencyLink{from: depending, to: dep}
	}()
}

// checkForCycle just checks to see if there's a dependency cycle. It doesn't compute the cycle to avoid excess
// allocations. buildCycle can be used to reconstruct the cycle once one has been found.
func (c *cycleDetector) checkForCycle(head, tail BuildLabel) bool {
	if tailDeps, ok := c.deps[tail]; ok {
		for _, dep := range tailDeps {
			if dep == head {
				// If the tail has a dependency on the head, we've found a cycle
				return true
			}

			if c.checkForCycle(head, dep) {
				return true
			}
		}
	}
	return false
}

// buildCycle is used to actually reconstruct the cycle after we've found one
func (c *cycleDetector) buildCycle(chain []BuildLabel) []BuildLabel {
	tail := chain[len(chain)-1]
	head := chain[0]

	if tailDeps, ok := c.deps[tail]; ok {
		for _, dep := range tailDeps {
			if dep == head {
				// If the tail has a dependency on the head, we've found a cycle
				return append(chain, dep)
			}

			if newChain := c.buildCycle(append(chain, dep)); newChain != nil {
				return newChain
			}
		}
	}
	return nil
}

func (c *cycleDetector) addDep(link dependencyLink) error {
	if c.checkForCycle(link.from, link.to) {
		return failWithGraphCycle(c.buildCycle([]BuildLabel{link.from, link.to}))
	}
	c.deps[link.from] = append(c.deps[link.from], link.to)
	return nil
}

func failWithGraphCycle(cycle dependencyChain) error {
	msg := "Dependency cycle found:\n"
	msg += cycle.String()
	return fmt.Errorf("%s \nSorry, but you'll have to refactor your build files to avoid this cycle.", msg)
}

func (c *cycleDetector) run() {
	go func() {
		for next := range c.queue {
			if err := c.addDep(next); err != nil {
				log.Fatalf("%v", err)
			}
		}
	}()
}
