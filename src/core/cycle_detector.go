package core

import "strings"

type dependencyChain []BuildLabel
type dependencyLink struct {
	label BuildLabel
	dep   BuildLabel
}

func (c dependencyChain) String() string {
	labels := make([]string, len(c))
	for i, l := range c {
		labels[i] = l.String()
	}
	return strings.Join(labels, "\n -> ")
}

type cycleDetector struct {
	deps  map[BuildLabel]BuildLabel
	queue chan dependencyLink
}

func newCycleDetector() *cycleDetector {
	c := new(cycleDetector)
	c.deps = map[BuildLabel]BuildLabel{}

	c.queue = make(chan dependencyLink, 100000)

	return c
}

// AddDependency queues up another dependency for the cycle detector to check
func (c *cycleDetector) AddDependency(depending BuildLabel, dep BuildLabel) {
	go func() { c.queue <- dependencyLink{label: depending, dep: dep} }()
}

func (c *cycleDetector) buildChain(chain dependencyChain) dependencyChain {
	if next, ok := c.deps[chain[len(chain)-1]]; ok {
		if next == chain[0] {
			return append(chain, next)
		}
		return c.buildChain(append(chain, next))
	}
	return nil
}

// TODO(jpoole) unit tests
func (c *cycleDetector) checkForCycle(dep, next BuildLabel) dependencyChain {
	return c.buildChain(dependencyChain{dep, next})
}

func (c *cycleDetector) addDep(depending BuildLabel, dep BuildLabel) {
	if cycle := c.checkForCycle(depending, dep); cycle != nil {
		failWithGraphCycle(cycle)
	}
	c.deps[depending] = dep
}

func failWithGraphCycle(cycle dependencyChain) {
	msg := "Dependency cycle found:\n"
	msg += cycle.String()
	log.Fatalf("%s \nSorry, but you'll have to refactor your build files to avoid this cycle.", msg)
}

func (c *cycleDetector) run() {
	go func() {
		for next := range c.queue {
			c.addDep(next.label, next.dep)
		}
	}()
}
