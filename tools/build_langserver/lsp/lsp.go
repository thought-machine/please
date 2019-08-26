// Package lsp implements the Language Server Protocol for Please BUILD files.
package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
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
	methods  map[string]method
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

type method struct {
	Func   reflect.Value
	Params reflect.Type
}

// NewHandler returns a new Handler.
func NewHandler() *Handler {
	h := &Handler{
		docs: map[string]*doc{},
		pkgs: &pkg{},
	}
	h.methods = map[string]method{
		"initialize":                  h.method(h.initialize),
		"initialized":                 h.method(h.initialized),
		"shutdown":                    h.method(h.shutdown),
		"exit":                        h.method(h.exit),
		"textDocument/didOpen":        h.method(h.didOpen),
		"textDocument/didChange":      h.method(h.didChange),
		"textDocument/didSave":        h.method(h.didSave),
		"textDocument/didClose":       h.method(h.didClose),
		"textDocument/formatting":     h.method(h.formatting),
		"textDocument/completion":     h.method(h.completion),
		"textDocument/documentSymbol": h.method(h.symbols),
		"textDocument/definition":     h.method(h.definition),
		"textDocument/declaration":    h.method(h.definition),
	}
	return h
}

// Handle implements the jsonrpc2.Handler interface
func (h *Handler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	if resp, err := h.handle(req.Method, req.Params); err != nil {
		if err := conn.ReplyWithError(ctx, req.ID, err); err != nil {
			log.Error("Failed to send error response: %s", err)
		}
	} else if resp != nil {
		if err := conn.Reply(ctx, req.ID, resp); err != nil {
			log.Error("Failed to send response: %s", err)
		}
	}
}

// handle is the slightly higher-level handler that deals with individual methods.
func (h *Handler) handle(method string, params *json.RawMessage) (i interface{}, err *jsonrpc2.Error) {
	start := time.Now()
	log.Debug("Received %s message", method)
	defer func() {
		if r := recover(); r != nil {
			log.Error("Panic in handler for %s: %s", method, r)
			err = &jsonrpc2.Error{
				Code:    jsonrpc2.CodeInternalError,
				Message: fmt.Sprintf("%s", r),
			}
		} else {
			log.Debug("Handled %s message in %s", method, time.Since(start))
		}
	}()
	m, present := h.methods[method]
	if !present {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeMethodNotFound}
	}
	p := reflect.New(m.Params)
	if err := json.Unmarshal(*params, p.Interface()); err != nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}
	ret := m.Func.Call([]reflect.Value{p.Elem()})
	if err, ok := ret[1].Interface().(error); ok && err != nil {
		log.Warning("Error from handler for %s: %s", method, err)
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInternalError, Message: err.Error()}
	} else if ret[0].IsNil() {
		return nil, nil
	}
	return ret[0].Interface(), nil
}

// method converts a function to a method struct
func (h *Handler) method(f interface{}) method {
	return method{
		Func:   reflect.ValueOf(f),
		Params: reflect.TypeOf(f).In(0),
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

func (h *Handler) initialized(params *struct{}) (*struct{}, error) {
	// Not doing anything here. Unsure right now what this is really for.
	return nil, nil
}

func (h *Handler) shutdown(params *struct{}) (*struct{}, error) {
	return nil, nil
}

func (h *Handler) exit(params *struct{}) (*struct{}, error) {
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
