package asp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thought-machine/please/src/core"
)

func TestFindTarget(t *testing.T) {
	state := core.NewDefaultBuildState()
	p := NewParser(state)
	stmts, err := p.ParseFileOnly("src/parse/asp/test_data/example.build")
	require.NoError(t, err)

	stmt := FindTarget(stmts, "asp")
	require.NotNil(t, stmt)
	assert.Equal(t, 1, stmt.Pos.Line)

	stmt = FindTarget(stmts, "parser_test")
	require.NotNil(t, stmt)
	assert.Equal(t, 16, stmt.Pos.Line)

	stmt = FindTarget(stmts, "lexer_test")
	require.NotNil(t, stmt)
	assert.Equal(t, 26, stmt.Pos.Line)

	stmt = FindTarget(stmts, "wibble")
	assert.Nil(t, stmt)
}

func TestGetExtents(t *testing.T) {
	state := core.NewDefaultBuildState()
	p := NewParser(state)
	stmts, err := p.ParseFileOnly("src/parse/asp/test_data/example.build")
	require.NoError(t, err)

	stmt := FindTarget(stmts, "asp")
	require.NotNil(t, stmt)
	begin, end := GetExtents(stmts, stmt, 200)
	assert.Equal(t, 1, begin)
	assert.Equal(t, 15, end)

	stmt = FindTarget(stmts, "parser_test")
	require.NotNil(t, stmt)
	begin, end = GetExtents(stmts, stmt, 200)
	assert.Equal(t, 16, begin)
	assert.Equal(t, 25, end)

	stmt = FindTarget(stmts, "lexer_test")
	require.NotNil(t, stmt)
	begin, end = GetExtents(stmts, stmt, 200)
	assert.Equal(t, 26, begin)
	assert.Equal(t, 35, end)

	stmt = FindTarget(stmts, "util_test")
	require.NotNil(t, stmt)
	begin, end = GetExtents(stmts, stmt, 200)
	assert.Equal(t, 36, begin)
	assert.Equal(t, 200, end)
}

func TestGetArg(t *testing.T) {
	state := core.NewDefaultBuildState()
	p := NewParser(state)
	stmts, err := p.ParseFileOnly("src/parse/asp/test_data/example.build")
	require.NoError(t, err)

	stmt := FindTarget(stmts, "asp")
	require.NotNil(t, stmt)
	vis := FindArgument(stmt, "visibility")
	require.NotNil(t, vis)
	assert.Equal(t, 13, vis.Value.Pos.Line)
	assert.Nil(t, FindArgument(stmt, "wibble"))
}
