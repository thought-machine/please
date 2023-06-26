package lsp

import (
	"testing"

	"github.com/sourcegraph/go-lsp"
	"github.com/stretchr/testify/assert"
)

const testURI = "file:///test_data/test.build"

func TestSymbols(t *testing.T) {
	h := initHandlerText(`"test"`)
	syms := []lsp.SymbolInformation{}
	err := h.Request("textDocument/documentSymbol", &lsp.DocumentSymbolParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: testURI,
		},
	}, &syms)
	assert.NoError(t, err)
	assert.Equal(t, []lsp.SymbolInformation{
		{
			Name:     "test",
			Kind:     lsp.SKString,
			Location: loc(0, 0, 0, 6),
		},
	}, syms)
}

func TestMoreSymbols(t *testing.T) {
	h := initHandlerText(`["a", 1, True, None]`)
	syms := []lsp.SymbolInformation{}
	err := h.Request("textDocument/documentSymbol", &lsp.DocumentSymbolParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: testURI,
		},
	}, &syms)
	assert.NoError(t, err)
	assert.Equal(t, []lsp.SymbolInformation{
		{
			Name:     "list",
			Kind:     lsp.SKArray,
			Location: loc(0, 0, 0, 20),
		},
		{
			Name:     "a",
			Kind:     lsp.SKString,
			Location: loc(0, 1, 0, 4),
		},
		{
			Name:     "1",
			Kind:     lsp.SKNumber,
			Location: loc(0, 6, 0, 7),
		},
		{
			Name:     "True",
			Kind:     lsp.SKBoolean,
			Location: loc(0, 9, 0, 13),
		},
		{
			Name:     "None",
			Kind:     lsp.SKConstant,
			Location: loc(0, 15, 0, 19),
		},
	}, syms)
}

func TestTopLevelFunctionSymbols(t *testing.T) {
	h := initHandlerText(`go_library(
    name = "lib",
    srcs = ["lib.go"],
)`)
	syms := []lsp.SymbolInformation{}
	err := h.Request("textDocument/documentSymbol", &lsp.DocumentSymbolParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: testURI,
		},
	}, &syms)
	assert.NoError(t, err)
	// TODO(peterebden): ideally this list should include the function arguments too
	assert.Equal(t, []lsp.SymbolInformation{
		{
			Name:     "go_library",
			Kind:     lsp.SKFunction,
			Location: loc(0, 0, 3, 1),
		},
		{
			Name:     "name",
			Kind:     lsp.SKKey,
			Location: loc(1, 4, 1, 8),
		},
		{
			Name:     "lib",
			Kind:     lsp.SKString,
			Location: loc(1, 11, 1, 16),
		},
		{
			Name:     "srcs",
			Kind:     lsp.SKKey,
			Location: loc(2, 4, 2, 8),
		},
		{
			Name:     "list",
			Kind:     lsp.SKArray,
			Location: loc(2, 11, 2, 21),
		},
		{
			Name:     "lib.go",
			Kind:     lsp.SKString,
			Location: loc(2, 12, 2, 20),
		},
	}, syms)
}

func initHandlerText(content string) *Handler {
	h := initHandler()
	err := h.Request("textDocument/didOpen", &lsp.DidOpenTextDocumentParams{
		TextDocument: lsp.TextDocumentItem{
			URI:  testURI,
			Text: content,
		},
	}, nil)
	if err != nil {
		panic(err)
	}
	return h
}

func xrng(startLine, startChar, endLine, endChar int) lsp.Range {
	return lsp.Range{
		Start: lsp.Position{Line: startLine, Character: startChar},
		End:   lsp.Position{Line: endLine, Character: endChar},
	}
}

func loc(startLine, startChar, endLine, endChar int) lsp.Location {
	return lsp.Location{
		URI:   testURI,
		Range: xrng(startLine, startChar, endLine, endChar),
	}
}
