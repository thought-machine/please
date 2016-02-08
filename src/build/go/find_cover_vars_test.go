package buildgo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The library here is a (very) reduced version of core that only has one file in it.
var coverageVars = []CoverVar{{
	Dir:     "src/build/go/test_data",
	Package: "core",
	Var:     "GoCover_lock_go",
	File:    "src/build/go/test_data/lock.go",
}}

func TestReadPkgdef(t *testing.T) {
	vars, err := readPkgdef("src/build/go/test_data/core.a")
	assert.NoError(t, err)
	assert.Equal(t, coverageVars, vars)
}

func TestReadCopiedPkgdef(t *testing.T) {
	// Sanity check that this file exists.
	vars, err := readPkgdef("src/build/go/test_data/x/core.a")
	assert.NoError(t, err)
	expected := []CoverVar{{
		Dir:     "src/build/go/test_data/x",
		Package: "core",
		Var:     "GoCover_lock_go",
		File:    "src/build/go/test_data/x/lock.go",
	}}
	assert.Equal(t, expected, vars)
}

func TestFindCoverVars(t *testing.T) {
	vars, err := FindCoverVars("src/build/go/test_data", []string{"src/build/go/test_data/x"})
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
