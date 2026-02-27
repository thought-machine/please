// Package lsp implements the Language Server Protocol for Please BUILD files.
package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/jsonrpc2"
	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/rules"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/parse"
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
	builtins map[string][]builtin
	pkgs     *pkg
	root     string
}

type builtin struct {
	Stmt        *asp.Statement
	Pos, EndPos asp.FilePosition
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
		docs:     map[string]*doc{},
		pkgs:     &pkg{},
		builtins: map[string][]builtin{},
	}
}

// Handle implements the jsonrpc2.Handler interface
func (h *Handler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	if resp, err := h.handle(req.Method, req.Params); err != nil {
		// check if the error is a jsonrpc error
		jsonerr, ok := err.(*jsonrpc2.Error)

		if !ok {
			// if it's not a jsonrpc error then create a CodeInternalError
			jsonerr = &jsonrpc2.Error{
				Code:    jsonrpc2.CodeInternalError,
				Message: err.Error(),
			}
		}

		if err := conn.ReplyWithError(ctx, req.ID, jsonerr); err != nil {
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
	config, err := core.ReadDefaultConfigFiles(fs.HostFS, nil)
	if err != nil {
		log.Error("Error reading configuration: %s", err)
		config = core.DefaultConfiguration()
	}
	h.state = core.NewBuildState(config)
	h.state.NeedBuild = false
	// Initialize the parser on state first, so that plz.RunHost uses the same parser.
	// This ensures plugin subincludes are stored in the same AST cache we use.
	parse.InitParser(h.state)
	h.parser = parse.GetAspParser(h.state)
	if h.parser == nil {
		return nil, fmt.Errorf("failed to get asp parser from state")
	}
	// Parse everything in the repo up front.
	// This is a lot easier than trying to do clever partial parses later on, although
	// eventually we may want that if we start dealing with truly large repos.
	go func() {
		// Start a goroutine to periodically load parser functions as they become available.
		// This allows go-to-definition to work progressively while the full parse runs.
		done := make(chan struct{})
		go func() {
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-done:
					h.loadParserFunctions()
					return
				case <-ticker.C:
					h.loadParserFunctions()
				}
			}
		}()
		plz.RunHost(core.WholeGraph, h.state)
		close(done)
		log.Debug("initial parse complete")
		h.buildPackageTree()
		log.Debug("built completion package tree")
	}()
	// Record all the builtin functions now
	if err := h.loadBuiltins(); err != nil {
		return nil, err
	}
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

// loadBuiltins extracts & loads all the builtin functions at startup.
func (h *Handler) loadBuiltins() error {
	assets, err := rules.AllAssets()
	if err != nil {
		return err
	}
	for _, asset := range assets {
		dir, err := os.UserCacheDir()
		if err != nil {
			return fmt.Errorf("Cannot determine user cache dir: %s", err)
		} else if err := os.MkdirAll(filepath.Join(dir, "please"), core.DirPermissions); err != nil {
			return fmt.Errorf("Cannot create cache dir: %s", err)
		}
		dest := filepath.Join(dir, "please", asset)
		data, err := rules.ReadAsset(asset)
		if err != nil {
			return fmt.Errorf("Failed to extract builtin rules for %s: %s", asset, err)
		}
		if err := os.WriteFile(dest, data, 0644); err != nil {
			return fmt.Errorf("Failed to extract builtin rules for %s: %s", asset, err)
		}
		stmts, err := h.parser.ParseFileOnly(dest)
		if err != nil {
			return fmt.Errorf("Failed to parse builtins: %s", err)
		}
		f := asp.NewFile(dest, data)
		for _, stmt := range stmts {
			if stmt.FuncDef != nil {
				h.builtins[stmt.FuncDef.Name] = append(h.builtins[stmt.FuncDef.Name], builtin{
					Stmt:   stmt,
					Pos:    f.Pos(stmt.Pos),
					EndPos: f.Pos(stmt.EndPos),
				})
			}
		}
	}
	log.Debug("loaded builtin function information")
	return nil
}

// loadParserFunctions loads function definitions from the parser's ASTs.
// This includes plugin-defined functions like go_library, python_library, etc.
func (h *Handler) loadParserFunctions() {
	funcsByFile := h.parser.AllFunctionsByFile()
	if funcsByFile == nil {
		return
	}
	h.mutex.Lock()
	defer h.mutex.Unlock()
	for filename, stmts := range funcsByFile {
		// Read the file to create a File object for position conversion
		data, err := os.ReadFile(filename)
		if err != nil {
			log.Warning("failed to read file %s: %v", filename, err)
			continue
		}
		file := asp.NewFile(filename, data)
		for _, stmt := range stmts {
			name := stmt.FuncDef.Name
			h.builtins[name] = append(h.builtins[name], builtin{
				Stmt:   stmt,
				Pos:    file.Pos(stmt.Pos),
				EndPos: file.Pos(stmt.EndPos),
			})
		}
	}
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
