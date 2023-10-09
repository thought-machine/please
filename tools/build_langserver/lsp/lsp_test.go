package lsp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bazelbuild/buildtools/build"
	"github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/jsonrpc2"
	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
)

func init() {
	cli.InitLogging(6)
}

func TestInitialize(t *testing.T) {
	h := NewHandler()
	result := &lsp.InitializeResult{}
	err := h.Request("initialize", &lsp.InitializeParams{
		Capabilities: lsp.ClientCapabilities{},
		RootURI:      lsp.DocumentURI("file://" + filepath.Join(os.Getenv("TEST_DIR"), "tools/build_langserver/lsp/test_data")),
	}, result)
	assert.NoError(t, err)
	assert.True(t, result.Capabilities.TextDocumentSync.Options.OpenClose)
	assert.Equal(t, filepath.Join(os.Getenv("TEST_DIR"), "tools/build_langserver/lsp/test_data"), h.root)
}

func TestInitializeNoURI(t *testing.T) {
	h := NewHandler()
	result := &lsp.InitializeResult{}
	err := h.Request("initialize", &lsp.InitializeParams{
		Capabilities: lsp.ClientCapabilities{},
	}, result)
	assert.Error(t, err)
}

const testContent = `
go_library(
    name = "lsp",
    srcs = ["lsp.go"],
    deps = [
        "//third_party/go:lsp",
    ],
)
`

const testContent2 = `
go_library(
    name = "lsp",
    srcs = ["lsp.go"],
    deps = [
        "//third_party/go:lsp",
    ],
)

go_test(
    name = "lsp_test",
    srcs = ["lsp_test.go"],
    deps = [
        ":lsp",
        "///third_party/go/github.com_stretchr_testify//assert",
    ],
)
`

func TestDidOpen(t *testing.T) {
	h := initHandler()
	err := h.Request("textDocument/didOpen", &lsp.DidOpenTextDocumentParams{
		TextDocument: lsp.TextDocumentItem{
			URI:  "file://test/test.build",
			Text: testContent,
		},
	}, nil)
	assert.NoError(t, err)
	assert.Equal(t, testContent, h.CurrentContent("test/test.build"))
}

func TestDidChange(t *testing.T) {
	// TODO(peterebden): change this when we support incremental changes.
	h := initHandler()
	err := h.Request("textDocument/didOpen", &lsp.DidOpenTextDocumentParams{
		TextDocument: lsp.TextDocumentItem{
			URI:  "file://test/test.build",
			Text: testContent,
		},
	}, nil)
	assert.NoError(t, err)
	err = h.Request("textDocument/didChange", &lsp.DidChangeTextDocumentParams{
		TextDocument: lsp.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: lsp.TextDocumentIdentifier{
				URI: "file://test/test.build",
			},
		},
		ContentChanges: []lsp.TextDocumentContentChangeEvent{
			{
				Text: testContent2,
			},
		},
	}, nil)
	assert.NoError(t, err)
	assert.Equal(t, testContent2, h.CurrentContent("test/test.build"))
}

func TestDidSave(t *testing.T) {
	h := initHandler()
	err := h.Request("textDocument/didOpen", &lsp.DidOpenTextDocumentParams{
		TextDocument: lsp.TextDocumentItem{
			URI:  "file://test/test.build",
			Text: testContent,
		},
	}, nil)
	assert.NoError(t, err)
	err = h.Request("textDocument/didSave", &lsp.DidSaveTextDocumentParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: "file://test/test.build",
		},
	}, nil)
	assert.NoError(t, err)
}

func TestDidClose(t *testing.T) {
	h := initHandler()
	err := h.Request("textDocument/didOpen", &lsp.DidOpenTextDocumentParams{
		TextDocument: lsp.TextDocumentItem{
			URI:  "file://test/test.build",
			Text: testContent,
		},
	}, nil)
	assert.NoError(t, err)
	err = h.Request("textDocument/didClose", &lsp.DidCloseTextDocumentParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: "file://test/test.build",
		},
	}, nil)
	assert.NoError(t, err)
}

const testFormattingContent = `go_test(
    name = "lsp_test",
    srcs = ["lsp_test.go"],
    deps = [":lsp","///third_party/go/github.com_stretchr_testify//assert"],
)
`

func TestFormatting(t *testing.T) {
	h := initHandler()
	err := h.Request("textDocument/didOpen", &lsp.DidOpenTextDocumentParams{
		TextDocument: lsp.TextDocumentItem{
			URI:  "file://test/test.build",
			Text: testFormattingContent,
		},
	}, nil)
	assert.NoError(t, err)
	edits := []lsp.TextEdit{}
	err = h.Request("textDocument/formatting", &lsp.DocumentFormattingParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: "file://test/test.build",
		},
	}, &edits)
	assert.NoError(t, err)
	assert.Equal(t, []lsp.TextEdit{
		{
			Range: lsp.Range{
				Start: lsp.Position{Line: 3, Character: 0},
				End:   lsp.Position{Line: 3, Character: 76},
			},
			NewText: `    deps = [`,
		},
		{
			Range: lsp.Range{
				Start: lsp.Position{Line: 4, Character: 0},
				End:   lsp.Position{Line: 4, Character: 1},
			},
			NewText: `        ":lsp",`,
		},
		{
			Range: lsp.Range{
				Start: lsp.Position{Line: 5, Character: 0},
				End:   lsp.Position{Line: 5, Character: 0},
			},
			NewText: `        "///third_party/go/github.com_stretchr_testify//assert",`,
		},
		{
			Range: lsp.Range{
				Start: lsp.Position{Line: 6, Character: 0},
				End:   lsp.Position{Line: 6, Character: 0},
			},
			NewText: "    ],\n)\n",
		},
	}, edits)
}

const testFormattingMalformedContent = `go_test(
    name = "lsp_test",
    srcs = ["lsp_test.go"]  # no comma
    deps = [":lsp","///third_party/go/github.com_stretchr_testify//assert"],
)
`

func TestFormattingMalformedContent(t *testing.T) {
	h := initHandler()
	err := h.Request("textDocument/didOpen", &lsp.DidOpenTextDocumentParams{
		TextDocument: lsp.TextDocumentItem{
			URI:  "file://test/test.build",
			Text: testFormattingMalformedContent,
		},
	}, nil)
	assert.NoError(t, err)
	edits := []lsp.TextEdit{}
	err = h.Request("textDocument/formatting", &lsp.DocumentFormattingParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: "file://test/test.build",
		},
	}, &edits)
	assert.Error(t, err)

	// check that it's a ParseError
	_, ok := err.(build.ParseError)
	assert.True(t, ok)
}

func TestShutdown(t *testing.T) {
	h := initHandler()
	r := h.Conn.(*rpc)
	err := h.Request("shutdown", &struct{}{}, nil)
	assert.NoError(t, err)
	// Shouldn't be closed yet
	assert.False(t, r.Closed)
	err = h.Request("exit", &struct{}{}, nil)
	assert.NoError(t, err)
	assert.True(t, r.Closed)
}

const testCompletionContent = `
go_library(
    name = "test",
    srcs = glob(["*.go"]),
    deps = [
        "//src/core:"
    ],
)
`

func TestCompletion(t *testing.T) {
	h := initHandler()
	err := h.Request("textDocument/didOpen", &lsp.DidOpenTextDocumentParams{
		TextDocument: lsp.TextDocumentItem{
			URI:  "file://test/test.build",
			Text: testCompletionContent,
		},
	}, nil)
	assert.NoError(t, err)
	h.WaitForPackage("src/core")
	completions := &lsp.CompletionList{}
	err = h.Request("textDocument/completion", &lsp.CompletionParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{
				URI: "file://test/test.build",
			},
			Position: lsp.Position{
				Line:      5,
				Character: 20,
			},
		},
	}, completions)
	assert.NoError(t, err)
	assert.Equal(t, &lsp.CompletionList{
		IsIncomplete: false,
		Items: []lsp.CompletionItem{
			{
				Label:            "//src/core:core",
				Kind:             lsp.CIKValue,
				InsertTextFormat: lsp.ITFPlainText,
				TextEdit:         textEdit("core", 5, 20),
			},
		},
	}, completions)
}

func TestCompletionPackages(t *testing.T) {
	h := initHandler()
	err := h.Request("textDocument/didOpen", &lsp.DidOpenTextDocumentParams{
		TextDocument: lsp.TextDocumentItem{
			URI:  "file://test/test.build",
			Text: testCompletionContent,
		},
	}, nil)
	assert.NoError(t, err)
	h.WaitForPackage("src/core")
	h.WaitForPackageTree()
	completions := &lsp.CompletionList{}
	err = h.Request("textDocument/completion", &lsp.CompletionParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{
				URI: "file://test/test.build",
			},
			Position: lsp.Position{
				Line:      5,
				Character: 12,
			},
		},
	}, completions)
	assert.NoError(t, err)
	assert.Equal(t, &lsp.CompletionList{
		IsIncomplete: true,
		Items: []lsp.CompletionItem{
			{
				Label:            "//src",
				Kind:             lsp.CIKValue,
				InsertTextFormat: lsp.ITFPlainText,
				TextEdit:         textEdit("rc", 5, 12),
			},
		},
	}, completions)
}

const testCompletionContentInMemory = `
go_library(
    name = "test",
    srcs = glob(["*.go"], exclude=["*_test.go"]),
    deps = [
        "//src/core"
    ],
)

go_test(
    name = "test_test",
    srcs = glob(["*_test.go"]),
    deps = [
        ":",
    ],
)
`

func TestCompletionInMemory(t *testing.T) {
	h := initHandler()
	err := h.Request("textDocument/didOpen", &lsp.DidOpenTextDocumentParams{
		TextDocument: lsp.TextDocumentItem{
			URI:  "file://test/test.build",
			Text: testCompletionContentInMemory,
		},
	}, nil)
	assert.NoError(t, err)
	h.WaitForPackage("src/core")
	h.WaitForPackageTree()
	completions := &lsp.CompletionList{}
	err = h.Request("textDocument/completion", &lsp.CompletionParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{
				URI: "file://test/test.build",
			},
			Position: lsp.Position{
				Line:      13,
				Character: 10,
			},
		},
	}, completions)
	assert.NoError(t, err)
	assert.Equal(t, &lsp.CompletionList{
		IsIncomplete: false,
		Items: []lsp.CompletionItem{
			{
				Label:            ":test",
				Kind:             lsp.CIKValue,
				InsertTextFormat: lsp.ITFPlainText,
				TextEdit:         textEdit("test", 13, 10),
			},
			// TODO(peterebden): We should filter this out really...
			{
				Label:            ":test_test",
				Kind:             lsp.CIKValue,
				InsertTextFormat: lsp.ITFPlainText,
				TextEdit:         textEdit("test_test", 13, 10),
			},
		},
	}, completions)
}

const testCompletionContentPartial = `
go_library(
    name = "test",
    srcs = glob(["*.go"]),
    deps = [
        "//src/core:
`

func TestCompletionPartial(t *testing.T) {
	h := initHandler()
	err := h.Request("textDocument/didOpen", &lsp.DidOpenTextDocumentParams{
		TextDocument: lsp.TextDocumentItem{
			URI:  "file://test/test.build",
			Text: testCompletionContentPartial,
		},
	}, nil)
	assert.NoError(t, err)
	h.WaitForPackage("src/core")
	completions := &lsp.CompletionList{}
	err = h.Request("textDocument/completion", &lsp.CompletionParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{
				URI: "file://test/test.build",
			},
			Position: lsp.Position{
				Line:      5,
				Character: 20,
			},
		},
	}, completions)
	assert.NoError(t, err)
	assert.Equal(t, &lsp.CompletionList{
		IsIncomplete: false,
		Items: []lsp.CompletionItem{
			{
				Label:            "//src/core:core",
				Kind:             lsp.CIKValue,
				InsertTextFormat: lsp.ITFPlainText,
				TextEdit:         textEdit("core", 5, 20),
			},
		},
	}, completions)
}

func TestCompletionFunction(t *testing.T) {
	h := initHandler()
	err := h.Request("textDocument/didOpen", &lsp.DidOpenTextDocumentParams{
		TextDocument: lsp.TextDocumentItem{
			URI:  testURI,
			Text: `plugin_repo()`,
		},
	}, nil)
	assert.NoError(t, err)
	completions := &lsp.CompletionList{}
	err = h.Request("textDocument/completion", &lsp.CompletionParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{
				URI: testURI,
			},
			Position: lsp.Position{
				Line:      0,
				Character: 4,
			},
		},
	}, completions)
	assert.NoError(t, err)
	assert.Equal(t, &lsp.CompletionList{
		IsIncomplete: false,
		Items: []lsp.CompletionItem{{
			Label:            "plugin_repo",
			Kind:             lsp.CIKFunction,
			InsertTextFormat: lsp.ITFPlainText,
			TextEdit:         textEdit("plugin_repo", 0, 4),
			Documentation:    h.builtins["plugin_repo"].Stmt.FuncDef.Docstring,
		}},
	}, completions)
}

func TestCompletionPartialFunction(t *testing.T) {
	h := initHandler()
	err := h.Request("textDocument/didOpen", &lsp.DidOpenTextDocumentParams{
		TextDocument: lsp.TextDocumentItem{
			URI:  testURI,
			Text: `plugin_re`,
		},
	}, nil)
	assert.NoError(t, err)
	completions := &lsp.CompletionList{}
	err = h.Request("textDocument/completion", &lsp.CompletionParams{
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{
				URI: testURI,
			},
			Position: lsp.Position{
				Line:      0,
				Character: 9,
			},
		},
	}, completions)
	assert.NoError(t, err)
	assert.Equal(t, &lsp.CompletionList{
		IsIncomplete: false,
		Items: []lsp.CompletionItem{{
			Label:            "plugin_repo",
			Kind:             lsp.CIKFunction,
			InsertTextFormat: lsp.ITFPlainText,
			TextEdit:         textEdit("plugin_repo", 0, 9),
			Documentation:    h.builtins["plugin_repo"].Stmt.FuncDef.Docstring,
		}},
	}, completions)
}

func textEdit(text string, line, col int) *lsp.TextEdit {
	return &lsp.TextEdit{
		NewText: text,
		Range: lsp.Range{
			Start: lsp.Position{Line: line, Character: col},
			End:   lsp.Position{Line: line, Character: col},
		},
	}
}

const testDiagnosticsContent = `
go_library(
    name = "test",
    srcs = glob(["*.go"]),
    deps = [
        "//src/core:core",
        "//src/core:config_test",
        "//src/core:nope",
    ],
)
`

func TestDiagnostics(t *testing.T) {
	h := initHandler()
	h.WaitForPackage("src/core")
	err := h.Request("textDocument/didOpen", &lsp.DidOpenTextDocumentParams{
		TextDocument: lsp.TextDocumentItem{
			URI:  "file://test/test.build",
			Text: testDiagnosticsContent,
		},
	}, nil)
	assert.NoError(t, err)
	r := h.Conn.(*rpc)
	msg := <-r.Notifications
	assert.Equal(t, "textDocument/publishDiagnostics", msg.Method)
	assert.Equal(t, &lsp.PublishDiagnosticsParams{
		URI: lsp.DocumentURI("file://" + filepath.Join(os.Getenv("TEST_DIR"), "tools/build_langserver/lsp/test_data/test/test.build")),
		Diagnostics: []lsp.Diagnostic{
			{
				Range: lsp.Range{
					Start: lsp.Position{Line: 6, Character: 9},
					End:   lsp.Position{Line: 6, Character: 32},
				},
				Severity: lsp.Error,
				Source:   "plz tool langserver",
				Message:  "Target //src/core:config_test is not visible to this package",
			},
			{
				Range: lsp.Range{
					Start: lsp.Position{Line: 7, Character: 9},
					End:   lsp.Position{Line: 7, Character: 25},
				},
				Severity: lsp.Error,
				Source:   "plz tool langserver",
				Message:  "Target //src/core:nope does not exist",
			},
		},
	}, msg.Payload)
}

// initHandler is a wrapper around creating a new handler and initializing it, which is
// more convenient for most tests.
func initHandler() *Handler {
	h := NewHandler()
	h.Conn = &rpc{
		Notifications: make(chan message, 100),
	}
	result := &lsp.InitializeResult{}
	if err := h.Request("initialize", &lsp.InitializeParams{
		Capabilities: lsp.ClientCapabilities{},
		RootURI:      lsp.DocumentURI("file://" + filepath.Join(os.Getenv("TEST_DIR"), "tools/build_langserver/lsp/test_data")),
	}, result); err != nil {
		log.Fatalf("init failed: %s", err)
	}
	return h
}

type message struct {
	Method  string
	Payload interface{}
}

type rpc struct {
	Closed        bool
	Notifications chan message
}

func (r *rpc) Close() error {
	r.Closed = true
	return nil
}

func (r *rpc) Notify(ctx context.Context, method string, params interface{}, opts ...jsonrpc2.CallOption) error {
	r.Notifications <- message{Method: method, Payload: params}
	return nil
}

// Request is a slightly higher-level wrapper for testing that handles JSON serialisation.
func (h *Handler) Request(method string, req, resp interface{}) error {
	b, err := json.Marshal(req)
	if err != nil {
		log.Fatalf("failed to encode request: %s", err)
	}
	msg := json.RawMessage(b)
	i, e := h.handle(method, &msg)
	if e != nil || resp == nil {
		return e
	}
	// serialise and deserialise, great...
	b, err = json.Marshal(i)
	if err != nil {
		log.Fatalf("failed to encode response: %s", err)
	} else if err := json.Unmarshal(b, resp); err != nil {
		log.Fatalf("failed to decode response: %s", err)
	}
	return e
}

// CurrentContent returns the current contents of a document.
func (h *Handler) CurrentContent(doc string) string {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	d := h.docs[doc]
	if d == nil {
		log.Error("unknown doc %s", doc)
		return ""
	}
	return strings.Join(d.Content, "\n")
}

// WaitForPackage blocks until the given package has been parsed.
func (h *Handler) WaitForPackage(pkg string) {
	for result := range h.state.Results() {
		if result.Status == core.PackageParsed && result.Label.PackageName == pkg {
			return
		} else if h.state.Graph.Package(pkg, "") != nil {
			return
		}
	}
}

// WaitForPackageTree blocks until the package tree is computed.
func (h *Handler) WaitForPackageTree() {
	// This is a bit yucky but there isn't any other way of syncing up to it.
	for h.pkgs.Subpackages == nil {
		time.Sleep(5 * time.Millisecond)
	}
}
