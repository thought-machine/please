// Package lsp implements the Language Server Protocol for Please BUILD files.
package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/jsonrpc2"
	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/help"
	"github.com/thought-machine/please/src/parse/asp"
	"github.com/thought-machine/please/src/plz"
)

var log = logging.MustGetLogger("lsp")

// A Handler is a handler suitable for use with jsonrpc2.
type Handler struct {
	Conn     Conn
	docs     map[string]*doc
	mutex    sync.Mutex // guards docs
	state    *core.BuildState
	parser   *asp.Parser
	builtins map[string]*asp.Statement
	pkgs     *pkg
	root     string
}

// A Conn is a minimal set of the jsonrpc2.Conn that we need.
type Conn interface {
	io.Closer
	// Notify sends an asynchronous notification.
	Notify(ctx context.Context, method string, params interface{}, opts ...jsonrpc2.CallOption) error
}

// NewHandler returns a new Handler.
func NewHandler() *Handler {
	return &Handler{
		docs: map[string]*doc{},
		pkgs: &pkg{},
	}
}

// Handle implements the jsonrpc2.Handler interface
func (h *Handler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	if resp, err := h.handle(req.Method, req.Params); err != nil {
		if err := conn.ReplyWithError(ctx, req.ID, err.(*jsonrpc2.Error)); err != nil {
			log.Error("Failed to send error response: %s", err)
		}
	} else if resp != nil {
		if err := conn.Reply(ctx, req.ID, resp); err != nil {
			log.Error("Failed to send response: %s", err)
		}
	}
}

// handle is the slightly higher-level handler that deals with individual methods.
func (h *Handler) handle(method string, params *json.RawMessage) (res interface{}, err error) {
	start := time.Now()
	log.Debug("Received %s message", method)
	defer func() {
		if r := recover(); r != nil {
			log.Error("Panic in handler for %s: %s", method, r)
			log.Debug("%s\n%v", r, string(debug.Stack()))
			err = &jsonrpc2.Error{
				Code:    jsonrpc2.CodeInternalError,
				Message: fmt.Sprintf("%s", r),
			}
		} else {
			log.Debug("Handled %s message in %s", method, time.Since(start))
		}
	}()

	switch method {
	case "initialize":
		initializeParams := &lsp.InitializeParams{}
		if err := json.Unmarshal(*params, initializeParams); err != nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}
		return h.initialize(initializeParams)
	case "initialized":
		// Not doing anything here. Unsure right now what this is really for.
		return nil, nil
	case "shutdown":
		return nil, nil
	case "exit":
		// exit is a request to terminate the process. We do this preferably by shutting
		// down the RPC connection but if we can't we just die.
		if h.Conn != nil {
			if err := h.Conn.Close(); err != nil {
				log.Fatalf("Failed to close connection: %s", err)
			}
		} else {
			log.Fatalf("No active connection to shut down")
		}
		return nil, nil
	case "textDocument/didOpen":
		didOpenParams := &lsp.DidOpenTextDocumentParams{}
		if err := json.Unmarshal(*params, didOpenParams); err != nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}
		return nil, h.didOpen(didOpenParams)
	case "textDocument/didChange":
		didChangeParams := &lsp.DidChangeTextDocumentParams{}
		if err := json.Unmarshal(*params, didChangeParams); err != nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}
		return nil, h.didChange(didChangeParams)
	case "textDocument/didSave":
		didSaveParams := &lsp.DidSaveTextDocumentParams{}
		if err := json.Unmarshal(*params, didSaveParams); err != nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}
		return nil, h.didSave(didSaveParams)
	case "textDocument/didClose":
		didCloseParams := &lsp.DidCloseTextDocumentParams{}
		if err := json.Unmarshal(*params, didCloseParams); err != nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}
		return nil, h.didClose(didCloseParams)
	case "textDocument/formatting":
		formattingParams := &lsp.DocumentFormattingParams{}
		if err := json.Unmarshal(*params, formattingParams); err != nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}
		return h.formatting(formattingParams)
	case "textDocument/completion":
		completionParams := &lsp.CompletionParams{}
		if err := json.Unmarshal(*params, completionParams); err != nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}
		return h.completion(completionParams)
	case "textDocument/documentSymbol":
		symbolParams := &lsp.DocumentSymbolParams{}
		if err := json.Unmarshal(*params, symbolParams); err != nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}
		return h.symbols(symbolParams)
	case "textDocument/declaration":
		fallthrough
	case "textDocument/definition":
		positionParams := &lsp.TextDocumentPositionParams{}
		if err := json.Unmarshal(*params, positionParams); err != nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}
		return h.definition(positionParams)
	default:
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeMethodNotFound}
	}
}

func (h *Handler) initialize(params *lsp.InitializeParams) (*lsp.InitializeResult, error) {
	// This is a bit yucky and stateful, but we only need to do it once.
	if err := os.Chdir(fromURI(params.RootURI)); err != nil {
		return nil, err
	}
	core.FindRepoRoot()
	h.root = core.RepoRoot
	if err := os.Chdir(h.root); err != nil {
		return nil, err
	}
	config, err := core.ReadDefaultConfigFiles(nil)
	if err != nil {
		log.Error("Error reading configuration: %s", err)
		config = core.DefaultConfiguration()
	}
	h.state = core.NewBuildState(config)
	h.state.NeedBuild = false
	// We need an unwrapped parser instance as well for raw access.
	h.parser = asp.NewParser(h.state)
	// Parse everything in the repo up front.
	// This is a lot easier than trying to do clever partial parses later on, although
	// eventually we may want that if we start dealing with truly large repos.
	go func() {
		plz.RunHost(core.WholeGraph, h.state)
		log.Debug("initial parse complete")
		h.buildPackageTree()
		log.Debug("built completion package tree")
	}()
	// Record all the builtin functions now
	h.builtins = help.AllBuiltinFunctions(h.state)
	return &lsp.InitializeResult{
		Capabilities: lsp.ServerCapabilities{
			TextDocumentSync: &lsp.TextDocumentSyncOptionsOrKind{
				Options: &lsp.TextDocumentSyncOptions{
					OpenClose: true,
					Change:    lsp.TDSKFull, // TODO(peterebden): Support incremental updates
				},
			},
			DocumentFormattingProvider: true,
			DocumentSymbolProvider:     true,
			DefinitionProvider:         true,
			CompletionProvider: &lsp.CompletionOptions{
				TriggerCharacters: []string{"/", ":"},
			},
		},
	}, nil
}

// fromURI converts a DocumentURI to a path.
func fromURI(uri lsp.DocumentURI) string {
	if !strings.HasPrefix(string(uri), "file://") {
		panic("invalid uri: " + uri)
	}
	return string(uri[7:])
}

// A Logger provides an interface to our logger.
type Logger struct{}

// Printf implements the jsonrpc2.Logger interface.
func (l Logger) Printf(tmpl string, args ...interface{}) {
	log.Info(tmpl, args...)
}
