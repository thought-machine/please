package output

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"core"
)

func TestFindGraphCycle(t *testing.T) {
	graph := core.NewGraph()
	graph.AddTarget(makeTarget("//src/output:target1", "//src/output:target2", "//src/output:target3", "//src/core:target2"))
	graph.AddTarget(makeTarget("//src/output:target2", "//src/output:target3", "//src/core:target1"))
	graph.AddTarget(makeTarget("//src/output:target3"))
	graph.AddTarget(makeTarget("//src/core:target1", "//third_party/go:target2", "//third_party/go:target3", "//src/core:target3"))
	graph.AddTarget(makeTarget("//src/core:target2", "//third_party/go:target3", "//src/output:target2"))
	graph.AddTarget(makeTarget("//src/core:target3", "//third_party/go:target2", "//src/output:target2"))
	graph.AddTarget(makeTarget("//third_party/go:target2", "//third_party/go:target1"))
	graph.AddTarget(makeTarget("//third_party/go:target3", "//third_party/go:target1"))
	graph.AddTarget(makeTarget("//third_party/go:target1"))
	updateDependencies(graph)

	cycle := findGraphCycle(graph, graph.TargetOrDie(core.ParseBuildLabel("//src/output:target1", "")))
	if len(cycle) == 0 {
		t.Fatalf("Failed to find cycle")
	} else if len(cycle) != 3 {
		t.Errorf("Found unexpected cycle of length %d", len(cycle))
	}
	assertTarget(t, cycle[0], "//src/output:target2")
	assertTarget(t, cycle[1], "//src/core:target1")
	assertTarget(t, cycle[2], "//src/core:target3")
}

func TestColouriseError(t *testing.T) {
	err := fmt.Errorf("/opt/tm/toolchains/1.8.2/usr/include/fst/label-reachable.h:176:39: error: non-const lvalue reference to type 'unordered_map<int, int>' cannot bind to a value of unrelated type 'unordered_map<unsigned char, unsigned char>'")
	expected := fmt.Errorf("\x1b[37;1m/opt/tm/toolchains/1.8.2/usr/include/fst/label-reachable.h, line 176, column 39:\x1b[0m \x1b[31;1merror: \x1b[0m\x1b[37;1mnon-const lvalue reference to type 'unordered_map<int, int>' cannot bind to a value of unrelated type 'unordered_map<unsigned char, unsigned char>'\x1b[0m")
	assert.EqualValues(t, expected, colouriseError(err))
}

// Factory function for build targets
func makeTarget(label string, deps ...string) *core.BuildTarget {
	target := core.NewBuildTarget(core.ParseBuildLabel(label, ""))
	for _, dep := range deps {
		target.AddDependency(core.ParseBuildLabel(dep, ""))
	}
	return target
}

// Set dependency pointers on all contents of the graph.
// Has to be done after to test cycles etc.
func updateDependencies(graph *core.BuildGraph) {
	for _, target := range graph.AllTargets() {
		for _, dep := range target.DeclaredDependencies() {
			graph.AddDependency(target.Label, dep)
		}
	}
}

func assertTarget(t *testing.T, target *core.BuildTarget, label string) {
	if target.Label != core.ParseBuildLabel(label, "") {
		t.Errorf("Unexpected target in detected cycle; expected %s, was %s", label, target.Label)
	}
}
