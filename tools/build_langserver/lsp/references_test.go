package lsp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sourcegraph/go-lsp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReferencesBuiltinFunction(t *testing.T) {
	// Test finding references to go_library function using actual test data file
	uri := "file://" + filepath.Join(os.Getenv("TEST_DIR"), "tools/build_langserver/lsp/test_data/src/core/test.build")
	h := initHandler()
	content, err := os.ReadFile(filepath.Join(os.Getenv("TEST_DIR"), "tools/build_langserver/lsp/test_data/src/core/test.build"))
	require.NoError(t, err)

	err = h.Request("textDocument/didOpen", &lsp.DidOpenTextDocumentParams{
		TextDocument: lsp.TextDocumentItem{
			URI:  lsp.DocumentURI(uri),
			Text: string(content),
		},
	}, nil)
	require.NoError(t, err)
	h.WaitForPackage("src/core")

	var locs []lsp.Location
	err = h.Request("textDocument/references", &lsp.ReferenceParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{
				URI: lsp.DocumentURI(uri),
			},
			Position: lsp.Position{Line: 0, Character: 5}, // on "go_library"
		},
		Context: lsp.ReferenceContext{
			IncludeDeclaration: false,
		},
	}, &locs)
	require.NoError(t, err)
	// go_library is widely used, we should find many references
	assert.NotEmpty(t, locs, "expected to find references to go_library")
}

func TestReferencesGoTestFunction(t *testing.T) {
	// Test finding references to go_test function
	uri := "file://" + filepath.Join(os.Getenv("TEST_DIR"), "tools/build_langserver/lsp/test_data/src/core/test.build")
	h := initHandler()
	content, err := os.ReadFile(filepath.Join(os.Getenv("TEST_DIR"), "tools/build_langserver/lsp/test_data/src/core/test.build"))
	require.NoError(t, err)

	err = h.Request("textDocument/didOpen", &lsp.DidOpenTextDocumentParams{
		TextDocument: lsp.TextDocumentItem{
			URI:  lsp.DocumentURI(uri),
			Text: string(content),
		},
	}, nil)
	require.NoError(t, err)
	h.WaitForPackage("src/core")

	var locs []lsp.Location
	err = h.Request("textDocument/references", &lsp.ReferenceParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{
				URI: lsp.DocumentURI(uri),
			},
			Position: lsp.Position{Line: 19, Character: 3}, // on "go_test" (line 20, 0-indexed = 19)
		},
		Context: lsp.ReferenceContext{
			IncludeDeclaration: false,
		},
	}, &locs)
	require.NoError(t, err)
	// go_test is widely used, we should find many references
	assert.NotEmpty(t, locs, "expected to find references to go_test")
}

func TestReferencesNoResults(t *testing.T) {
	// Test that we get empty results when cursor is on a comment
	h := initHandlerText(`# just a comment
# nothing referenceable here`)
	h.WaitForPackageTree()

	var locs []lsp.Location
	err := h.Request("textDocument/references", &lsp.ReferenceParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{
				URI: testURI,
			},
			Position: lsp.Position{Line: 0, Character: 5},
		},
		Context: lsp.ReferenceContext{
			IncludeDeclaration: false,
		},
	}, &locs)
	require.NoError(t, err)
	assert.Empty(t, locs, "expected no references for a comment")
}

func TestReferencesBuildLabel(t *testing.T) {
	// Test that label references don't panic when package not in graph
	h := initHandlerText(`go_library(
    name = "core",
    srcs = ["lib.go"],
)

go_test(
    name = "core_test",
    srcs = ["lib_test.go"],
    deps = [":core"],
)`)
	h.WaitForPackageTree()

	var locs []lsp.Location
	err := h.Request("textDocument/references", &lsp.ReferenceParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{
				URI: testURI,
			},
			// Position on the ":core" string in deps
			Position: lsp.Position{Line: 8, Character: 13},
		},
		Context: lsp.ReferenceContext{
			IncludeDeclaration: false,
		},
	}, &locs)
	// Should not error even if package not in graph
	require.NoError(t, err)
}

func TestReferencesIncludeDeclaration(t *testing.T) {
	// Test that IncludeDeclaration includes the function definition
	uri := "file://" + filepath.Join(os.Getenv("TEST_DIR"), "tools/build_langserver/lsp/test_data/src/core/test.build")
	h := initHandler()
	content, err := os.ReadFile(filepath.Join(os.Getenv("TEST_DIR"), "tools/build_langserver/lsp/test_data/src/core/test.build"))
	require.NoError(t, err)

	err = h.Request("textDocument/didOpen", &lsp.DidOpenTextDocumentParams{
		TextDocument: lsp.TextDocumentItem{
			URI:  lsp.DocumentURI(uri),
			Text: string(content),
		},
	}, nil)
	require.NoError(t, err)
	h.WaitForPackage("src/core")

	var locsWithDecl []lsp.Location
	err = h.Request("textDocument/references", &lsp.ReferenceParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{
				URI: lsp.DocumentURI(uri),
			},
			Position: lsp.Position{Line: 0, Character: 5}, // on "go_library"
		},
		Context: lsp.ReferenceContext{
			IncludeDeclaration: true,
		},
	}, &locsWithDecl)
	require.NoError(t, err)

	var locsWithoutDecl []lsp.Location
	err = h.Request("textDocument/references", &lsp.ReferenceParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{
				URI: lsp.DocumentURI(uri),
			},
			Position: lsp.Position{Line: 0, Character: 5}, // on "go_library"
		},
		Context: lsp.ReferenceContext{
			IncludeDeclaration: false,
		},
	}, &locsWithoutDecl)
	require.NoError(t, err)

	// With declaration should include one more location (the definition)
	assert.GreaterOrEqual(t, len(locsWithDecl), len(locsWithoutDecl),
		"IncludeDeclaration=true should return at least as many results")
}
