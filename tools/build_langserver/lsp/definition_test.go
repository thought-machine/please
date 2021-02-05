package lsp

import (
	"os"
	"path"
	"testing"

	"github.com/sourcegraph/go-lsp"
	"github.com/stretchr/testify/assert"
)

func TestDefinition(t *testing.T) {
	// the x= construction is a bit weird but works with a definition that's an expression.
	h := initHandlerText(`x = go_bindata`)
	h.WaitForPackageTree()
	locs := []lsp.Location{}
	err := h.Request("textDocument/definition", &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: testURI,
		},
		Position: lsp.Position{Line: 0, Character: 5},
	}, &locs)
	assert.NoError(t, err)
	assert.Equal(t, []lsp.Location{
		{
			URI:   lsp.DocumentURI("file://" + path.Join(os.Getenv("TEST_DIR"), "tools/build_langserver/lsp/test_data/build_defs/go_bindata.build_defs")),
			Range: xrng(0, 0, 22, 5),
		},
	}, locs)
}

func TestDefinitionStatement(t *testing.T) {
	h := initHandlerText(`go_bindata()`)
	h.WaitForPackageTree()
	locs := []lsp.Location{}
	err := h.Request("textDocument/definition", &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: testURI,
		},
		Position: lsp.Position{Line: 0, Character: 5},
	}, &locs)
	assert.NoError(t, err)
	assert.Equal(t, []lsp.Location{
		{
			URI:   lsp.DocumentURI("file://" + path.Join(os.Getenv("TEST_DIR"), "tools/build_langserver/lsp/test_data/build_defs/go_bindata.build_defs")),
			Range: xrng(0, 0, 22, 5),
		},
	}, locs)
}

func TestDefinitionBuiltin(t *testing.T) {
	h := initHandlerText(`genrule()`)
	h.WaitForPackageTree()
	locs := []lsp.Location{}
	err := h.Request("textDocument/definition", &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: testURI,
		},
		Position: lsp.Position{Line: 0, Character: 5},
	}, &locs)
	assert.NoError(t, err)
	cacheDir, _ := os.UserCacheDir()
	assert.Equal(t, []lsp.Location{
		{
			URI:   lsp.DocumentURI("file://" + path.Join(cacheDir, "please/misc_rules.build_defs")),
			Range: xrng(3, 0, 140, 5),
		},
	}, locs)
}
