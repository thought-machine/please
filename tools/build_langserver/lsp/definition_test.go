package lsp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sourcegraph/go-lsp"
	"github.com/stretchr/testify/assert"
)

func TestDefinition(t *testing.T) {
	// the x= construction is a bit weird but works with a definition that's an expression.
	h := initHandlerText(`x = go_bindata`)
	h.WaitForPackageTree()
	var locs []lsp.Location
	err := h.Request("textDocument/definition", &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: testURI,
		},
		Position: lsp.Position{Line: 0, Character: 5},
	}, &locs)
	assert.NoError(t, err)
	assert.Equal(t, []lsp.Location{
		{
			URI:   lsp.DocumentURI("file://" + filepath.Join(os.Getenv("TEST_DIR"), "tools/build_langserver/lsp/test_data/build_defs/go_bindata.build_defs")),
			Range: xrng(0, 0, 22, 5),
		},
	}, locs)
}

func TestDefinitionStatement(t *testing.T) {
	h := initHandlerText(`go_bindata()`)
	h.WaitForPackageTree()
	var locs []lsp.Location
	err := h.Request("textDocument/definition", &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: testURI,
		},
		Position: lsp.Position{Line: 0, Character: 5},
	}, &locs)
	assert.NoError(t, err)
	assert.Equal(t, []lsp.Location{
		{
			URI:   lsp.DocumentURI("file://" + filepath.Join(os.Getenv("TEST_DIR"), "tools/build_langserver/lsp/test_data/build_defs/go_bindata.build_defs")),
			Range: xrng(0, 0, 22, 5),
		},
	}, locs)
}

func TestDefinitionBuiltin(t *testing.T) {
	h := initHandlerText(`genrule()`)
	h.WaitForPackageTree()
	var locs []lsp.Location
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
			URI:   lsp.DocumentURI("file://" + filepath.Join(cacheDir, "please/misc_rules.build_defs")),
			Range: xrng(3, 0, 144, 5),
		},
	}, locs)
}

func TestDefinitionBuildLabel(t *testing.T) {
	h := initHandlerText("go_bindata(\n    deps = ['//src/core'],\n)")
	h.WaitForPackageTree()
	var locs []lsp.Location
	err := h.Request("textDocument/definition", &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: testURI,
		},
		Position: lsp.Position{Line: 1, Character: 15},
	}, &locs)
	assert.NoError(t, err)
	assert.Equal(t, []lsp.Location{
		{
			URI:   lsp.DocumentURI("file://" + filepath.Join(os.Getenv("TEST_DIR"), "tools/build_langserver/lsp/test_data/src/core/test.build")),
			Range: xrng(1, 1, 17, 2),
		},
	}, locs)
}

func TestDefinitionFileInput(t *testing.T) {
	content := `go_test(
    name = "config_test",
    srcs = ["config_test.go"],
)`
	uri := "file://" + filepath.Join(os.Getenv("TEST_DIR"), "tools/build_langserver/lsp/test_data/src/core/test.build")
	h := initHandler()
	err := h.Request("textDocument/didOpen", &lsp.DidOpenTextDocumentParams{
		TextDocument: lsp.TextDocumentItem{
			URI:  lsp.DocumentURI(uri),
			Text: content,
		},
	}, nil)
	assert.NoError(t, err)
	h.WaitForPackage("src/core")

	var locs []lsp.Location
	err = h.Request("textDocument/definition", &lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: lsp.DocumentURI(uri),
		},
		Position: lsp.Position{Line: 2, Character: 17},
	}, &locs)
	assert.NoError(t, err)
	assert.Equal(t, []lsp.Location{
		{
			URI:   lsp.DocumentURI("file://" + filepath.Join(os.Getenv("TEST_DIR"), "tools/build_langserver/lsp/test_data/src/core/config_test.go")),
			Range: xrng(0, 0, 0, 0),
		},
	}, locs)
}
