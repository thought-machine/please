package buildgo

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseTestSources(t *testing.T) {
	descr, err := parseTestSources([]string{"src/build/go/test_data/example_test.go"})
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
	assert.Equal(t, functions, descr.Functions)
}

func TestParseTestSourcesWithMain(t *testing.T) {
	descr, err := parseTestSources([]string{"src/build/go/test_data/example_test_main.go"})
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
	assert.Equal(t, functions, descr.Functions)
}

func TestParseTestSourcesFailsGracefully(t *testing.T) {
	_, err := parseTestSources([]string{"wibble"})
	assert.Error(t, err)
}

func TestWriteTestMain(t *testing.T) {
	err := WriteTestMain(
		[]string{"src/build/go/test_data/example_test.go"},
		"test.go",
		[]CoverVar{},
	)
	assert.NoError(t, err)
	// It's not really practical to assert the contents of the file in great detail.
	// We'll do the obvious thing of asserting that it is valid Go source.
	f, err := parser.ParseFile(token.NewFileSet(), "test.go", nil, 0)
	assert.NoError(t, err)
	assert.Equal(t, "main", f.Name.Name)
}

func TestWriteTestMainWithCoverage(t *testing.T) {
	err := WriteTestMain(
		[]string{"src/build/go/test_data/example_test.go"},
		"test.go",
		[]CoverVar{{
			Dir:        "src/build/go/test_data",
			Package:    "core",
			ImportPath: "core",
			Var:        "GoCover_lock_go",
			File:       "src/build/go/test_data/lock.go",
		}},
	)
	assert.NoError(t, err)
	// It's not really practical to assert the contents of the file in great detail.
	// We'll do the obvious thing of asserting that it is valid Go source.
	f, err := parser.ParseFile(token.NewFileSet(), "test.go", nil, 0)
	assert.NoError(t, err)
	assert.Equal(t, "main", f.Name.Name)
}

func TestExtraImportPaths(t *testing.T) {
	assert.Equal(t, extraImportPaths("core", []CoverVar{
		{ImportPath: "core"},
		{ImportPath: "output"},
	}), []string{
		"\"core\"",
		"_cover0 \"core\"",
		"_cover1 \"output\"",
	})
}
