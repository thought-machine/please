package test

import (
	"go/parser"
	"go/token"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseTestSources(t *testing.T) {
	descr, err := parseTestSources([]string{"tools/please_go/test/test_data/test/example_test.go"})
	assert.NoError(t, err)
	assert.Equal(t, "buildgo", descr.Package)
	assert.Equal(t, "", descr.Main)
	functions := []string{
		"TestReadPkgdef",
		"TestReadCopiedPkgdef",
		"TestFindCoverVars",
		"TestFindCoverVarsFailsGracefully",
		"TestFindCoverVarsReturnsNothingForEmptyPath",
	}
	assert.Equal(t, functions, descr.TestFunctions)
}

func TestParseTestSourcesWithMain(t *testing.T) {
	descr, err := parseTestSources([]string{"tools/please_go/test/test_data/main/example_test_main.go"})
	assert.NoError(t, err)
	assert.Equal(t, "parse", descr.Package)
	assert.Equal(t, "TestMain", descr.Main)
	functions := []string{
		"TestParseSourceBuildLabel",
		"TestParseSourceRelativeBuildLabel",
		"TestParseSourceFromSubdirectory",
		"TestParseSourceFromOwnedSubdirectory",
		"TestParseSourceWithParentPath",
		"TestParseSourceWithAbsolutePath",
		"TestAddTarget",
	}
	assert.Equal(t, functions, descr.TestFunctions)
}

func TestParseTestSourcesFailsGracefully(t *testing.T) {
	_, err := parseTestSources([]string{"wibble"})
	assert.Error(t, err)
}

func TestWriteTestMain(t *testing.T) {
	err := WriteTestMain("test_pkg", []string{"tools/please_go/test/test_data/test/example_test.go"}, "test.go", false, []CoverVar{}, false, true)
	assert.NoError(t, err)
	// It's not really practical to assert the contents of the file in great detail.
	// We'll do the obvious thing of asserting that it is valid Go source.
	f, err := parser.ParseFile(token.NewFileSet(), "test.go", nil, 0)
	assert.NoError(t, err)
	assert.Equal(t, "main", f.Name.Name)
}

func TestWriteTestMainWithCoverage(t *testing.T) {
	err := WriteTestMain("test_package", []string{"tools/please_go/test/test_data/test/example_test.go"}, "test.go", true, []CoverVar{{
		Dir:        "tools/please_go/test/test_data",
		ImportPath: "core",
		Var:        "GoCover_lock_go",
		File:       "tools/please_go/test/test_data/lock.go",
	}}, false, false)
	assert.NoError(t, err)
	// It's not really practical to assert the contents of the file in great detail.
	// We'll do the obvious thing of asserting that it is valid Go source.
	f, err := parser.ParseFile(token.NewFileSet(), "test.go", nil, 0)
	assert.NoError(t, err)
	assert.Equal(t, "main", f.Name.Name)
}

func TestWriteTestMainWithBenchmark(t *testing.T) {
	err := WriteTestMain("test_package", []string{"tools/please_go/test/test_data/bench/example_benchmark_test.go"}, "test.go", true, []CoverVar{}, true, true)
	assert.NoError(t, err)
	// It's not really practical to assert the contents of the file in great detail.
	// We'll do the obvious thing of asserting that it is valid Go source.
	f, err := parser.ParseFile(token.NewFileSet(), "test.go", nil, 0)
	assert.NoError(t, err)
	assert.Equal(t, "main", f.Name.Name)

	test, err := os.ReadFile("test.go")
	assert.NoError(t, err)
	assert.Contains(t, string(test), "BenchmarkExample")
}

func TestExtraImportPaths(t *testing.T) {
	assert.Equal(t, extraImportPaths("core", "core", []CoverVar{
		{ImportPath: "core"},
		{ImportPath: "output"},
	}), []string{
		"core \"core\"",
		"_cover0 \"core\"",
		"_cover1 \"output\"",
	})
}

func TestExtraImportPathsWithImportPath(t *testing.T) {
	assert.Equal(t, extraImportPaths("core", "core", []CoverVar{
		{ImportPath: "github.com/thought-machine/please/src/core"},
		{ImportPath: "github.com/thought-machine/please/output"},
	}), []string{
		"core \"core\"",
		"_cover0 \"github.com/thought-machine/please/src/core\"",
		"_cover1 \"github.com/thought-machine/please/output\"",
	})
}
