package langserver

import (
	"context"
	"core"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"tools/build_langserver/lsp"

	"github.com/sourcegraph/jsonrpc2"
	"gopkg.in/op/go-logging.v1"
)

var log = logging.MustGetLogger("lsp")

// NewHandler creates a BUILD file language server handler
func NewHandler() jsonrpc2.Handler {
	h := &LsHandler{
		IsServerDown: false,
	}
	return langHandler{
		jsonrpc2.HandlerWithError(h.Handle)}
}

// handler wraps around LsHandler to correctly handler requests in the correct order
type langHandler struct {
	jsonrpc2.Handler
}

// LsHandler is the main handler struct of the language server handler
type LsHandler struct {
	init     *lsp.InitializeParams
	analyzer *Analyzer
	mu       sync.Mutex
	conn     *jsonrpc2.Conn

	workspace *workspaceStore

	repoRoot     string
	requestStore *requestStore

	IsServerDown         bool
	supportedCompletions []lsp.CompletionItemKind
}

// Handle function takes care of all the incoming from the client, and returns the correct response
func (h *LsHandler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Method != "initialize" && h.init == nil {
		return nil, fmt.Errorf("server must be initialized")
	}
	h.conn = conn

	log.Info(fmt.Sprintf("handling method %s with params: %s", req.Method, req.Params))
	methods := map[string]func(ctx context.Context, req *jsonrpc2.Request) (result interface{}, err error){
		"initialize":              h.handleInit,
		"initialzed":              h.handleInitialized,
		"shutdown":                h.handleShutDown,
		"exit":                    h.handleExit,
		"$/cancelRequest":         h.handleCancel,
		"textDocument/hover":      h.handleHover,
		"textDocument/completion": h.handleCompletion,
	}

	if req.Method != "initialize" && req.Method != "exit" &&
		req.Method != "initialzed" && req.Method != "shutdown" {
		ctx = h.requestStore.Store(ctx, req)
		defer h.requestStore.Cancel(req.ID)
	}

	if method, ok := methods[req.Method]; ok {
		return method(ctx, req)
	}
	// TODO(bnm): call fs request handlers like, textDocument/didOpen

	return h.handleTDRequests(ctx, req)
}

func (h *LsHandler) handleInit(ctx context.Context, req *jsonrpc2.Request) (result interface{}, err error) {
	if h.init != nil {
		return nil, errors.New("language server is already initialized")
	}
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params lsp.InitializeParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	// Set the Init state of the handler
	h.mu.Lock()
	// TODO(bnmetrics): Ideas: this could essentially be a bit fragile.
	// maybe we can defer until user send a request with first file URL
	core.FindRepoRoot()

	// TODO(bnm): remove stuff with reporoot
	params.EnsureRoot()
	h.repoRoot = string(params.RootURI)
	h.workspace = newWorkspaceStore(params.RootURI)

	h.supportedCompletions = params.Capabilities.TextDocument.Completion.CompletionItemKind.ValueSet
	h.init = &params

	h.analyzer, err = newAnalyzer()
	if err != nil {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeParseError,
			Message: fmt.Sprintf("error in parsing .plzconfig file: %s", err),
		}
	}

	// Reset the requestStore, and get sub-context based on request ID
	reqStore := newRequestStore()
	h.requestStore = reqStore
	ctx = h.requestStore.Store(ctx, req)

	h.mu.Unlock()
	defer h.requestStore.Cancel(req.ID)

	// Fill in the response results
	TDsync := lsp.SyncIncremental
	completeOps := &lsp.CompletionOptions{
		ResolveProvider:   false,
		TriggerCharacters: []string{".", "//", ":"},
	}

	sigHelpOps := &lsp.SignatureHelpOptions{
		TriggerCharacters: []string{"{", ","},
	}

	defer log.Info("Plz build file language server initialized")
	return lsp.InitializeResult{
		Capabilities: lsp.ServerCapabilities{
			TextDocumentSync:           &TDsync,
			HoverProvider:              true,
			CompletionProvider:         completeOps,
			SignatureHelpProvider:      sigHelpOps,
			DefinitionProvider:         true,
			TypeDefinitionProvider:     true,
			ImplementationProvider:     true,
			ReferenceProvider:          true,
			DocumentFormattingProvider: true,
			DocumentHighlightProvider:  true,
			DocumentSymbolProvider:     true,
		},
	}, nil
}

func (h *LsHandler) handleInitialized(ctx context.Context, req *jsonrpc2.Request) (result interface{}, err error) {
	return nil, nil
}

func (h *LsHandler) handleShutDown(ctx context.Context, req *jsonrpc2.Request) (result interface{}, err error) {
	h.mu.Lock()
	if h.IsServerDown {
		log.Warning("Server is already down!")
	}
	h.IsServerDown = true
	defer h.mu.Unlock()
	return nil, nil
}

func (h *LsHandler) handleExit(ctx context.Context, req *jsonrpc2.Request) (result interface{}, err error) {
	h.handleShutDown(ctx, req)
	h.conn.Close()
	return nil, nil
}

func (h *LsHandler) handleCancel(ctx context.Context, req *jsonrpc2.Request) (result interface{}, err error) {
	// Is there is no param with Id, or if there is no requests stored currently, return nothing
	if req.Params == nil || h.requestStore.IsEmpty() {
		return nil, nil
	}

	var params lsp.CancelParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, &jsonrpc2.Error{
			Code:    lsp.RequestCancelled,
			Message: fmt.Sprintf("Cancellation of request(id: %s) failed", req.ID),
		}
	}

	defer h.requestStore.Cancel(params.ID)

	return nil, nil
}

func getURIAndHandleErrors(uri lsp.DocumentURI, method string) (lsp.DocumentURI, error) {
	documentURI, err := EnsureURL(uri, "file")
	if err != nil {
		message := fmt.Sprintf("invalid documentURI '%s' for method %s", documentURI, method)
		log.Error(message)
		return "", &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: message,
		}
	}
	return documentURI, err
}
