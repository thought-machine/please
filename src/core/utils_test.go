package core

import (
	"crypto/sha1"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollapseHash(t *testing.T) {
	// Test that these two come out differently
	input1 := [sha1.Size * 4]byte{}
	input2 := [sha1.Size * 4]byte{}
	for i := 0; i < sha1.Size; i++ {
		input1[i] = byte(i)
		input2[i] = byte(i * 2)
	}
	output1 := CollapseHash(input1[:])
	output2 := CollapseHash(input2[:])
	assert.NotEqual(t, output1, output2)
}

func TestCollapseHash2(t *testing.T) {
	// Test of a couple of cases that weren't different...
	input1, err1 := base64.URLEncoding.DecodeString("mByUsoTswXV2X_W6FHhBwJUCQM-YHJSyhOzBdXZf9boUeEHAlQJAz-DzaA7MCXxt5_FFws2WO51vKlqt-JThKzdEQn_bghpDDCuKOI9qGNI=")
	input2, err2 := base64.URLEncoding.DecodeString("rSH0PS_dftB6KN_Jnu_jszhbxiutIfQ9L91-0Hoo38me7-OzOFvGK-DzaA7MCXxt5_FFws2WO51vKlqt-JThKzdEQn_bghpDDCuKOI9qGNI=")
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	output1 := CollapseHash(input1)
	output2 := CollapseHash(input2)
	assert.NotEqual(t, output1, output2)
}

func TestIterSources(t *testing.T) {
	graph := buildGraph()
	iterSources := func(label string) []sourcePair {
		return toSlice(IterSources(graph, graph.TargetOrDie(ParseBuildLabel(label, ""))))
	}

	assert.Equal(t, []sourcePair{
		{"src/core/target1.go", "plz-out/tmp/src/core/target1#.build/src/core/target1.go"},
	}, iterSources("//src/core:target1"))

	assert.Equal(t, []sourcePair{
		{"src/core/target2.go", "plz-out/tmp/src/core/target2#.build/src/core/target2.go"},
		{"plz-out/gen/src/core/target1.a", "plz-out/tmp/src/core/target2#.build/src/core/target1.a"},
	}, iterSources("//src/core:target2"))

	assert.Equal(t, []sourcePair{
		{"src/build/target1.go", "plz-out/tmp/src/build/target1#.build/src/build/target1.go"},
		{"plz-out/gen/src/core/target1.a", "plz-out/tmp/src/build/target1#.build/src/core/target1.a"},
	}, iterSources("//src/build:target1"))

	assert.Equal(t, []sourcePair{
		{"src/output/output1.go", "plz-out/tmp/src/output/output1#.build/src/output/output1.go"},
		{"plz-out/gen/src/build/target1.a", "plz-out/tmp/src/output/output1#.build/src/build/target1.a"},
	}, iterSources("//src/output:output1"))

	assert.Equal(t, []sourcePair{
		{"src/output/output1.go", "plz-out/tmp/src/output/output1#.build/src/output/output1.go"},
		{"plz-out/gen/src/build/target1.a", "plz-out/tmp/src/output/output1#.build/src/build/target1.a"},
	}, iterSources("//src/output:output1"))

	assert.Equal(t, []sourcePair{
		{"src/output/output2.go", "plz-out/tmp/src/output/output2#.build/src/output/output2.go"},
		{"plz-out/gen/src/core/target2.a", "plz-out/tmp/src/output/output2#.build/src/core/target2.a"},
		{"plz-out/gen/src/output/output1.a", "plz-out/tmp/src/output/output2#.build/src/output/output1.a"},
	}, iterSources("//src/output:output2"))

	assert.Equal(t, []sourcePair{
		{"src/parse/target1.go", "plz-out/tmp/src/parse/target1#.build/src/parse/target1.go"},
		{"plz-out/gen/src/core/target2.a", "plz-out/tmp/src/parse/target1#.build/src/core/target2.a"},
		{"plz-out/gen/src/core/target1.a", "plz-out/tmp/src/parse/target1#.build/src/core/target1.a"},
	}, iterSources("//src/parse:target1"))

	assert.Equal(t, []sourcePair{
		{"src/parse/target2.go", "plz-out/tmp/src/parse/target2#.build/src/parse/target2.go"},
		{"plz-out/gen/src/parse/target1.a", "plz-out/tmp/src/parse/target2#.build/src/parse/target1.a"},
	}, iterSources("//src/parse:target2"))
}

func TestInitialPackageSimple(t *testing.T) {
	initialPackage = "src/core"
	p := InitialPackage()
	assert.Equal(t, []BuildLabel{{PackageName: "src/core", Name: "..."}}, p)
}

func TestInitialPackageIllegalLabel(t *testing.T) {
	// Moves up a directory because the last component isn't a legal package name.
	// This is not that common but does make our existing test work at least :)
	initialPackage = "plz-out/tmp/test/query_alltargets_test#.test"
	p := InitialPackage()
	assert.Equal(t, []BuildLabel{{PackageName: "plz-out/tmp/test", Name: "..."}}, p)
}

func TestInitialPackageRoot(t *testing.T) {
	// Test that we don't get stuck in an infinite loop or do anything similarly weird
	// when the input is empty.
	initialPackage = ""
	p := InitialPackage()
	assert.Equal(t, []BuildLabel{{PackageName: "", Name: "..."}}, p)
}

func TestInitialPackageUpToRoot(t *testing.T) {
	// Similar to above but when we don't start out at the root but back up to it.
	initialPackage = "query_alltargets_test#.test"
	p := InitialPackage()
	assert.Equal(t, []BuildLabel{{PackageName: "", Name: "..."}}, p)
}

// buildGraph builds a test graph which we use to test IterSources etc.
func buildGraph() *BuildGraph {
	graph := NewGraph()
	mt := func(label string, deps ...string) *BuildTarget {
		target := makeTarget(graph, label, deps...)
		graph.AddTarget(target)
		return target
	}

	mt("//src/core:target1")
	mt("//src/core:target2", "//src/core:target1")
	mt("//src/build:target1", "//src/core:target1")
	mt("//src/output:output1", "//src/build:target1")
	mt("//src/output:output2", "//src/output:output1", "//src/core:target2")
	t1 := mt("//src/parse:target1", "//src/core:target2")
	t1.NeedsTransitiveDependencies = true
	t1.OutputIsComplete = true
	mt("//src/parse:target2", "//src/parse:target1")

	return graph
}

// makeTarget creates a new build target for us.
func makeTarget(graph *BuildGraph, label string, deps ...string) *BuildTarget {
	target := NewBuildTarget(ParseBuildLabel(label, ""))
	for _, dep := range deps {
		t := graph.TargetOrDie(ParseBuildLabel(dep, ""))
		target.AddDependency(t.Label)
		target.resolveDependency(target.Label, t)
	}
	target.Sources = append(target.Sources, FileLabel{
		File:    target.Label.Name + ".go",
		Package: target.Label.PackageName,
	})
	target.AddOutput(target.Label.Name + ".a")
	return target
}

func toSlice(ch <-chan sourcePair) []sourcePair {
	ret := []sourcePair{}
	for x := range ch {
		ret = append(ret, x)
	}
	return ret
}
