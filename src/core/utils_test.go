package core

import (
	"crypto/sha1"
	"encoding/base64"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"cli"
)

var tenSecondsTime = 10 * time.Second
var tenSeconds = cli.Duration(tenSecondsTime)

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
	iterSources := func(label string) []SourcePair {
		return toSlice(IterSources(graph, graph.TargetOrDie(ParseBuildLabel(label, ""))))
	}

	assert.Equal(t, []SourcePair{
		{"src/core/target1.go", "plz-out/tmp/src/core/target1._build/src/core/target1.go"},
	}, iterSources("//src/core:target1"))

	assert.Equal(t, []SourcePair{
		{"src/core/target2.go", "plz-out/tmp/src/core/target2._build/src/core/target2.go"},
		{"plz-out/gen/src/core/target1.a", "plz-out/tmp/src/core/target2._build/src/core/target1.a"},
	}, iterSources("//src/core:target2"))

	assert.Equal(t, []SourcePair{
		{"src/build/target1.go", "plz-out/tmp/src/build/target1._build/src/build/target1.go"},
		{"plz-out/gen/src/core/target1.a", "plz-out/tmp/src/build/target1._build/src/core/target1.a"},
	}, iterSources("//src/build:target1"))

	assert.Equal(t, []SourcePair{
		{"src/output/output1.go", "plz-out/tmp/src/output/output1._build/src/output/output1.go"},
		{"plz-out/gen/src/build/target1.a", "plz-out/tmp/src/output/output1._build/src/build/target1.a"},
	}, iterSources("//src/output:output1"))

	assert.Equal(t, []SourcePair{
		{"src/output/output1.go", "plz-out/tmp/src/output/output1._build/src/output/output1.go"},
		{"plz-out/gen/src/build/target1.a", "plz-out/tmp/src/output/output1._build/src/build/target1.a"},
	}, iterSources("//src/output:output1"))

	assert.Equal(t, []SourcePair{
		{"src/output/output2.go", "plz-out/tmp/src/output/output2._build/src/output/output2.go"},
		{"plz-out/gen/src/core/target2.a", "plz-out/tmp/src/output/output2._build/src/core/target2.a"},
		{"plz-out/gen/src/output/output1.a", "plz-out/tmp/src/output/output2._build/src/output/output1.a"},
	}, iterSources("//src/output:output2"))

	assert.Equal(t, []SourcePair{
		{"src/parse/target1.go", "plz-out/tmp/src/parse/target1._build/src/parse/target1.go"},
		{"plz-out/gen/src/core/target2.a", "plz-out/tmp/src/parse/target1._build/src/core/target2.a"},
		{"plz-out/gen/src/core/target1.a", "plz-out/tmp/src/parse/target1._build/src/core/target1.a"},
	}, iterSources("//src/parse:target1"))

	assert.Equal(t, []SourcePair{
		{"src/parse/target2.go", "plz-out/tmp/src/parse/target2._build/src/parse/target2.go"},
		{"plz-out/gen/src/parse/target1.a", "plz-out/tmp/src/parse/target2._build/src/parse/target1.a"},
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
	initialPackage = "plz-out/tmp/test/query_alltargets_test._test"
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
	initialPackage = "query_alltargets_test._test"
	p := InitialPackage()
	assert.Equal(t, []BuildLabel{{PackageName: "", Name: "..."}}, p)
}

func TestLookPath(t *testing.T) {
	// Assume this will be present on the path somewhere (you've really got to have bash for plz)
	path, err := LookPath("bash", []string{"/usr/local/bin", "/usr/bin", "/bin"})
	assert.NoError(t, err)
	assert.Contains(t, []string{"/usr/local/bin/bash", "/usr/bin/bash", "/bin/bash"}, path)
	info, err := os.Stat(path)
	assert.NoError(t, err)
	assert.Equal(t, "bash", info.Name())
}

func TestLookPathColons(t *testing.T) {
	// We support having colons inside the path elements because people might find that more natural.
	path, err := LookPath("bash", []string{"/usr/local/bin:/usr/bin:/bin"})
	assert.NoError(t, err)
	assert.Contains(t, []string{"/usr/local/bin/bash", "/usr/bin/bash", "/bin/bash"}, path)
	info, err := os.Stat(path)
	assert.NoError(t, err)
	assert.Equal(t, "bash", info.Name())
}

func TestLookPathDoesntExist(t *testing.T) {
	_, err := LookPath("wibblewobbleflibble", []string{"/usr/local/bin", "/usr/bin", "/bin"})
	assert.Error(t, err)
}

func TestExecWithTimeout(t *testing.T) {
	out, err := ExecWithTimeoutSimple(tenSeconds, "true")
	assert.NoError(t, err)
	assert.Equal(t, 0, len(out))
}

func TestExecWithTimeoutFailure(t *testing.T) {
	out, err := ExecWithTimeoutSimple(tenSeconds, "false")
	assert.Error(t, err)
	assert.Equal(t, 0, len(out))
}

func TestExecWithTimeoutDeadline(t *testing.T) {
	out, err := ExecWithTimeoutSimple(cli.Duration(1*time.Nanosecond), "sleep", "10")
	assert.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "Timeout exceeded"))
	assert.Equal(t, 0, len(out))
}

func TestExecWithTimeoutOutput(t *testing.T) {
	state := NewDefaultBuildState()
	out, stderr, err := ExecWithTimeoutShell(state, nil, "", nil, tenSecondsTime, tenSeconds, false, "echo hello", false)
	assert.NoError(t, err)
	assert.Equal(t, "hello\n", string(out))
	assert.Equal(t, "hello\n", string(stderr))
}

func TestExecWithTimeoutStderr(t *testing.T) {
	state := NewDefaultBuildState()
	out, stderr, err := ExecWithTimeoutShell(state, nil, "", nil, tenSecondsTime, tenSeconds, false, "echo hello 1>&2", false)
	assert.NoError(t, err)
	assert.Equal(t, "", string(out))
	assert.Equal(t, "hello\n", string(stderr))
}

func TestAsyncDeleteDir(t *testing.T) {
	err := os.MkdirAll("test_dir/a/b/c", DirPermissions)
	assert.NoError(t, err)
	err = AsyncDeleteDir("test_dir")
	assert.NoError(t, err)
	for i := 0; i < 100; i++ {
		if !PathExists("test_dir") {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	assert.False(t, PathExists("test_dir"))
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

func toSlice(ch <-chan SourcePair) []SourcePair {
	ret := []SourcePair{}
	for x := range ch {
		ret = append(ret, x)
	}
	return ret
}
