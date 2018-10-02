package langserver

import (
	"context"
	"core"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/sourcegraph/jsonrpc2"
	"gopkg.in/op/go-logging.v1"

	"tools/build_langserver/lsp"
)

var log = logging.MustGetLogger("lsp")

// NewHandler creates a BUILD file language server handler
func NewHandler() jsonrpc2.Handler {
	return handler{jsonrpc2.HandlerWithError((&LsHandler{
		IsServerDown: false,
	}).Handle)}
}

// handler wraps around LsHandler to correctly handler requests in the correct order
type handler struct {
	jsonrpc2.Handler
}

// LsHandler is the main handler struct of the language server handler
type LsHandler struct {
	init *lsp.InitializeParams
	mu   sync.Mutex
	conn *jsonrpc2.Conn

	repoRoot     string
	requestStore *requestStore

	IsServerDown         bool
	supportedCompletions []lsp.CompletionItemKind
}

// Handle function takes care of all the incoming from the client, and returns the correct response
func (h *LsHandler) Handle(ctx context.Context, conn *jsonrpc2.Conn, request *jsonrpc2.Request) (result interface{}, err error) {
	if request.Method != "initialize" && h.init == nil {
		return nil, fmt.Errorf("server must be initialized")
	}
	h.conn = conn

	methods := map[string]func(ctx context.Context, request *jsonrpc2.Request) (result interface{}, err error){
		"initialize":      h.handleInit,
		"initialzed":      h.handleInitialized,
		"shutdown":        h.handleShutDown,
		"exit":            h.handleExit,
		"$/cancelRequest": h.handleCancel,
	}

	return methods[request.Method](ctx, request)

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
	// TODO(bnmetrics): Ideas: this could essentially  be a bit fragile.
	// maybe we can defer until user send a request with first file URL
	core.FindRepoRoot()

	h.repoRoot = core.RepoRoot
	h.supportedCompletions = params.Capabilities.TextDocument.Completion.CompletionItemKind.ValueSet

	params.EnsureRoot()
	h.init = &params

	// Reset the requestStore, and get sub-context based on request ID
	reqStore := requestStore{
		requests: make(map[jsonrpc2.ID]request),
	}
	h.requestStore = &reqStore
	ctx = h.requestStore.Store(ctx, req)

	h.mu.Unlock()

	defer h.requestStore.Cancel(req.ID)

	// Fill in the response results
	TDsync := lsp.SyncIncremental
	completeOps := &lsp.CompletionOptions{
		ResolveProvider:   true,
		TriggerCharacters: []string{"."},
	}

	sigHelpOps := &lsp.SignatureHelpOptions{
		TriggerCharacters: []string{"{", ","},
	}

	log.Info("Initialize plz build file language server...")
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

func (h *LsHandler) handleInitialized(ctx context.Context, request *jsonrpc2.Request) (result interface{}, err error) {
	return nil, nil
}

func (h *LsHandler) handleShutDown(ctx context.Context, request *jsonrpc2.Request) (result interface{}, err error) {
	h.mu.Lock()
	if h.IsServerDown {
		log.Warning("Server is already down!")
	}
	h.IsServerDown = true
	defer h.mu.Unlock()
	return nil, nil
}

func (h *LsHandler) handleExit(ctx context.Context, request *jsonrpc2.Request) (result interface{}, err error) {
	h.handleShutDown(ctx, request)
	h.conn.Close()
	return nil, nil
}

func (h *LsHandler) handleCancel(ctx context.Context, request *jsonrpc2.Request) (result interface{}, err error) {
	// Is there is no param with Id, or if there is no requests stored currently, return nothing
	if request.Params == nil || len(h.requestStore.requests) == 0 {
		return nil, nil
	}

	var params lsp.CancelParams
	if err := json.Unmarshal(*request.Params, &params); err != nil {
		return nil, &jsonrpc2.Error{
			Code:    lsp.RequestCancelled,
			Message: fmt.Sprintf("Cancellation of request(id: %s) failed", request.ID),
		}
	}

	h.requestStore.Cancel(params.ID)

	return nil, nil
}
