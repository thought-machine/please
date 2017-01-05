package buildgo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The library here is a (very) reduced version of core that only has one file in it.
var coverageVars = []CoverVar{{
	Dir:        "tools/please_go_test/test_data",
	ImportPath: "tools/please_go_test/test_data/core",
	Package:    "core",
	Var:        "GoCover_lock_go",
	File:       "tools/please_go_test/test_data/lock.go",
}}

func TestReadPkgdef(t *testing.T) {
	vars, err := readPkgdef("tools/please_go_test/test_data/core.a")
	assert.NoError(t, err)
	assert.Equal(t, coverageVars, vars)
}

func TestReadCopiedPkgdef(t *testing.T) {
	// Sanity check that this file exists.
	vars, err := readPkgdef("tools/please_go_test/test_data/x/core.a")
	assert.NoError(t, err)
	expected := []CoverVar{{
		Dir:        "tools/please_go_test/test_data/x",
		ImportPath: "tools/please_go_test/test_data/x/core",
		Package:    "core",
		Var:        "GoCover_lock_go",
		File:       "tools/please_go_test/test_data/x/lock.go",
	}}
	assert.Equal(t, expected, vars)
}

func TestFindCoverVars(t *testing.T) {
	vars, err := FindCoverVars("tools/please_go_test/test_data", []string{"tools/please_go_test/test_data/x", "tools/please_go_test/test_data/binary"})
	assert.NoError(t, err)
	assert.Equal(t, coverageVars, vars)
}

func TestFindCoverVarsFailsGracefully(t *testing.T) {
	_, err := FindCoverVars("wibble", []string{})
	assert.Error(t, err)
}

func TestFindCoverVarsReturnsNothingForEmptyPath(t *testing.T) {
	vars, err := FindCoverVars("", []string{})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(vars))
}

func TestFindBinaryCoverVars(t *testing.T) {
	// Test for Go 1.7 binary format.
	expected := []CoverVar{{
		Dir:        "tools/please_go_test/test_data/binary",
		ImportPath: "tools/please_go_test/test_data/binary/core",
		Package:    "core",
		Var:        "GoCover_lock_go",
		File:       "tools/please_go_test/test_data/binary/lock.go",
	}}
	vars, err := FindCoverVars("tools/please_go_test/test_data/binary", nil)
	assert.NoError(t, err)
	assert.Equal(t, expected, vars)
}
