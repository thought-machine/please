package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The library here is a (very) reduced version of core that only has one file in it.
var coverageVars = []CoverVar{{
	Dir:        "tools/please_go/test/test_data",
	ImportPath: "tools/please_go/test/test_data/core",
	Var:        "GoCover_lock_go",
	File:       "tools/please_go/test/test_data/lock.go",
}}

func TestFindCoverVars(t *testing.T) {
	vars, err := FindCoverVars("tools/please_go/test/test_data", "", "", false, []string{"tools/please_go/test/test_data/x", "tools/please_go/test/test_data/binary"}, nil)
	assert.NoError(t, err)
	assert.Equal(t, coverageVars, vars)
}

func TestFindCoverVarsFailsGracefully(t *testing.T) {
	_, err := FindCoverVars("wibble", "", "", false, nil, nil)
	assert.Error(t, err)
}

func TestFindCoverVarsReturnsNothingForEmptyPath(t *testing.T) {
	vars, err := FindCoverVars("", "", "", false, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(vars))
}

func TestFindBinaryCoverVars(t *testing.T) {
	// Test for Go 1.7 binary format.
	expected := []CoverVar{{
		Dir:        "tools/please_go/test/test_data/binary",
		ImportPath: "tools/please_go/test/test_data/binary/core",
		Var:        "GoCover_lock_go",
		File:       "tools/please_go/test/test_data/binary/lock.go",
	}}
	vars, err := FindCoverVars("tools/please_go/test/test_data/binary", "", "", false, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, expected, vars)
}

func TestFindCoverVarsExcludesSrcs(t *testing.T) {
	vars, err := FindCoverVars("tools/please_go/test/test_data/binary", "", "", false, nil, []string{"tools/please_go/test/test_data/binary/lock.go"})
	assert.NoError(t, err)
	var expected []CoverVar
	assert.Equal(t, expected, vars)
}
