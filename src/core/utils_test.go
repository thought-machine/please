package core

import (
	"crypto/sha1"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thought-machine/please/src/fs"
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
	state := NewDefaultBuildState()
	graph := buildGraph()

	type SourcePair struct{ Src, Tmp string }
	iterSources := func(label string) []SourcePair {
		ret := []SourcePair{}
		for src, tmp := range IterSources(state, graph, graph.TargetOrDie(ParseBuildLabel(label, "")), false) {
			ret = append(ret, SourcePair{src, tmp})
		}
		return ret
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
		{"plz-out/gen/src/build/target3.a", "plz-out/tmp/src/parse/target1._build/src/build/target3.a"},
		{"plz-out/gen/src/core/target2.a", "plz-out/tmp/src/parse/target1._build/src/core/target2.a"},
		{"plz-out/gen/src/core/target1.a", "plz-out/tmp/src/parse/target1._build/src/core/target1.a"},
	}, iterSources("//src/parse:target1"))

	assert.Equal(t, []SourcePair{
		{"src/parse/target2.go", "plz-out/tmp/src/parse/target2._build/src/parse/target2.go"},
		{"plz-out/gen/src/parse/target1.a", "plz-out/tmp/src/parse/target2._build/src/parse/target1.a"},
	}, iterSources("//src/parse:target2"))
}

func TestInitialPackageSimple(t *testing.T) {
	InitialPackagePath = "src/core"
	p := InitialPackage()
	assert.Equal(t, []BuildLabel{{PackageName: "src/core", Name: "..."}}, p)
}

func TestInitialPackageIllegalLabel(t *testing.T) {
	// Moves up a directory because the last component isn't a legal package name.
	// This is not that common but does make our existing test work at least :)
	InitialPackagePath = "plz-out/tmp/test/query_alltargets_test._test"
	p := InitialPackage()
	assert.Equal(t, []BuildLabel{{PackageName: "plz-out/tmp/test", Name: "..."}}, p)
}

func TestInitialPackageRoot(t *testing.T) {
	// Test that we don't get stuck in an infinite loop or do anything similarly weird
	// when the input is empty.
	InitialPackagePath = ""
	p := InitialPackage()
	assert.Equal(t, []BuildLabel{{PackageName: "", Name: "..."}}, p)
}

func TestInitialPackageUpToRoot(t *testing.T) {
	// Similar to above but when we don't start out at the root but back up to it.
	InitialPackagePath = "query_alltargets_test._test"
	p := InitialPackage()
	assert.Equal(t, []BuildLabel{{PackageName: "", Name: "..."}}, p)
}

// writeFakeTool creates an executable named tool, plus whatever extension the platform needs
// to consider it one, in a new directory, and returns the directory and the full path.
func writeFakeTool(t *testing.T, tool string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	file := filepath.Join(dir, tool+fs.ExeSuffix)
	require.NoError(t, os.WriteFile(file, nil, 0o755))
	return dir, file
}

func TestLookPath(t *testing.T) {
	// A tool we put there ourselves, rather than something the host is assumed to have: the
	// directories Please looks in by default differ per platform, and on Windows there are none.
	dir, file := writeFakeTool(t, "plz_look_path_test")
	found, err := LookPath("plz_look_path_test", []string{filepath.Join(dir, "nonexistent"), dir})
	require.NoError(t, err)
	assert.Equal(t, file, found)
}

func TestLookPathColons(t *testing.T) {
	// We support having the list separator inside the path elements because people might find
	// that more natural.
	dir, file := writeFakeTool(t, "plz_look_path_test")
	joined := strings.Join([]string{filepath.Join(dir, "nonexistent"), dir}, string(os.PathListSeparator))
	found, err := LookPath("plz_look_path_test", []string{joined})
	require.NoError(t, err)
	assert.Equal(t, file, found)
}

func TestLookPathDoesntExist(t *testing.T) {
	dir, _ := writeFakeTool(t, "plz_look_path_test")
	_, err := LookPath("wibblewobbleflibble", []string{dir, t.TempDir()})
	assert.Error(t, err)
	assert.NotContains(t, err.Error(), "No [build] path", "shouldn't advise configuring a path that is configured")
}

func TestLookPathWithNothingConfigured(t *testing.T) {
	// Only Please's own directory to search, which is what a Windows user gets before they set
	// [build] path - there is no default one there. Say so rather than just naming the one
	// directory we looked in.
	_, err := LookPath("wibblewobbleflibble", []string{t.TempDir()})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "No [build] path is configured")
}

// buildGraph builds a test graph which we use to test IterSources etc.
func buildGraph() *BuildGraph {
	graph := NewGraph()
	mt := func(label string, deps ...string) *BuildTarget {
		target := makeTarget4(graph, label, deps...)
		graph.AddTarget(target)
		return target
	}

	mt("//src/core:target1")
	mt("//src/core:target2", "//src/core:target1")
	mt("//src/build:target1", "//src/core:target1")
	mt("//src/output:output1", "//src/build:target1")
	mt("//src/output:output2", "//src/output:output1", "//src/core:target2")
	mt("//src/build:target3").AddSource(ParseBuildLabel("//src/core:target2", ""))
	t1 := mt("//src/parse:target1", "//src/build:target3", "//src/core:target2")
	t1.NeedsTransitiveDependencies = true
	t1.OutputIsComplete = true
	mt("//src/parse:target2", "//src/parse:target1")

	return graph
}

// makeTarget4 creates a new build target for us.
func makeTarget4(graph *BuildGraph, label string, deps ...string) *BuildTarget {
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

func TestFindRepoRootFromTerminatesAtTheRoot(t *testing.T) {
	// A walk that reaches the top of the filesystem without finding a marker has to stop.
	// On Windows it used not to: trimming the separator off "C:\" leaves "C:", and splitting
	// that returns it unchanged, so this spun for ever at one stat per iteration and every plz
	// run outside a repo hung instead of reporting that it could not find a root.
	wd, err := os.Getwd()
	require.NoError(t, err)
	root := filepath.VolumeName(wd) + string(filepath.Separator)

	done := make(chan struct{})
	go func() {
		defer close(done)
		dir, pkg := findRepoRootFrom(root, "a_file_that_is_not_there_"+t.Name())
		assert.Empty(t, dir)
		assert.Empty(t, pkg)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		// Failing rather than hanging the whole package, which is what this used to do.
		t.Fatal("findRepoRootFrom did not terminate at the filesystem root")
	}
}

func TestFindRepoRootFromFindsTheMarker(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "some", "package")
	require.NoError(t, os.MkdirAll(nested, os.ModeDir|0755))
	marker := "marker_" + t.Name()
	require.NoError(t, os.WriteFile(filepath.Join(root, marker), nil, 0644))

	dir, pkg := findRepoRootFrom(nested, marker)
	assert.Equal(t, root, dir)
	// Slash-separated whatever the OS gave us, because it is a package name.
	assert.Equal(t, "some/package", pkg)
}

func TestIsInRepoRoot(t *testing.T) {
	// The comparison this replaces was a plain HasPrefix against RepoRoot, which is in the
	// OS's own separator. Paths that arrive from outside - a file:// URL, a coverage report
	// from another tool - are slash-separated, so on Windows it never matched and the guard
	// that uses it silently stopped guarding.
	old := RepoRoot
	defer func() { RepoRoot = old }()
	RepoRoot = filepath.Join(string(filepath.Separator)+"home", "user", "repo")
	slashed := filepath.ToSlash(RepoRoot)

	assert.True(t, IsInRepoRoot(RepoRoot))
	assert.True(t, IsInRepoRoot(slashed), "a slash-separated path inside the repo is inside it")
	assert.True(t, IsInRepoRoot(slashed+"/src/core/utils.go"))
	assert.True(t, IsInRepoRoot(filepath.Join(RepoRoot, "src", "core")))

	assert.False(t, IsInRepoRoot(slashed+"sitory/src"), "only matches at a path boundary")
	assert.False(t, IsInRepoRoot("/somewhere/else"))
	assert.False(t, IsInRepoRoot(""))
}

func TestTrimRepoRoot(t *testing.T) {
	old := RepoRoot
	defer func() { RepoRoot = old }()
	RepoRoot = filepath.Join(string(filepath.Separator)+"home", "user", "repo")
	slashed := filepath.ToSlash(RepoRoot)

	assert.Equal(t, "src/core", TrimRepoRoot(slashed+"/src/core"))
	assert.Equal(t, filepath.Join("src", "core"), TrimRepoRoot(filepath.Join(RepoRoot, "src", "core")))
	// Left alone rather than mangled when it isn't ours.
	assert.Equal(t, "/somewhere/else", TrimRepoRoot("/somewhere/else"))
}
