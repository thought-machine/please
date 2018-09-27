package langerver

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/sourcegraph/jsonrpc2"
	"tools/build_langserver/lsp"
)


func NewHandler() jsonrpc2.Handler {
	// TODO: need to rethink this
	return jsonrpc2.HandlerWithError(LsHandler{
		IsServerDown:false,
	}.Handle)
}

type LsHandler struct {
	init *lsp.InitializeParams
	mu sync.Mutex
	conn *jsonrpc2.Conn


	IsServerDown bool
	SupportedCompletions []lsp.CompletionItemKind
}

func (h *LsHandler) Handle (ctx context.Context, conn *jsonrpc2.Conn, request *jsonrpc2.Request) (result interface{}, err error) {
	if request.Method != "initialize" && h.init == nil {
		return nil, errors.New("server must be initialized")
	}
	h.conn = conn

	// TODO: Select different methods based on request.Method

	methods := map[string]func(request *jsonrpc2.Request) (result interface{}, err error){
		"initialize": h.handleInit,
		"initialzed": h.handleInitialized,
		"shutdown": h.handleShutDown,
		"exit": h.handleExit,

	}

	methods[request.Method](request)


}

func (h *LsHandler) handleInit(request *jsonrpc2.Request) (result interface{}, err error) {
	if h.init != nil {
		return nil, errors.New("language server is already initialized")
	}
	if request.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params lsp.InitializeParams
	if err := json.Unmarshal(*request.Params, &params); err != nil {
		return nil, err
	}

	// Set the Init state of the handler
	h.mu.Lock()
	h.SupportedCompletions = params.Capabilities.TextDocument.Completion.CompletionItemKind.ValueSet
	params.EnsureRoot()
	h.init = &params

	h.mu.Unlock()


	// Fill in the response results
	TDsync := lsp.SyncIncremental
	completeOps := &lsp.CompletionOptions{
		ResolveProvider:true,
		TriggerCharacters:[]string{"."},
	}

	sigHelpOps := &lsp.SignatureHelpOptions{
		TriggerCharacters:[]string{"{", ","},
	}

	log.Info("Initialize plz build file language server...")
	return lsp.InitializeResult{
		Capabilities: lsp.ServerCapabilities{
			TextDocumentSync:&TDsync,
			HoverProvider:true,
			CompletionProvider: completeOps,
			SignatureHelpProvider:sigHelpOps,
			DefinitionProvider:true,
			TypeDefinitionProvider:true,
			ImplementationProvider:true,
			ReferenceProvider:true,
			DocumentFormattingProvider:true,
			DocumentHighlightProvider:true,
			DocumentSymbolProvider:true,
		},
	}, nil
}

func (h *LsHandler) handleInitialized(request *jsonrpc2.Request) (result interface{}, err error) {
	if h.init != nil {
		return nil, nil
	}
}


func (h *LsHandler) handleShutDown(request *jsonrpc2.Request) (result interface{}, err error) {
	h.mu.Lock()
	if h.IsServerDown {
		log.Warning("Server is already down!")
	}
	h.IsServerDown = true
	h.mu.Lock()
	return nil, nil
}

func (h *LsHandler) handleExit(request *jsonrpc2.Request) (result interface{}, err error) {
	h.handleShutDown(request)
	h.conn.Close()
	return nil, nil
}

func (h *LsHandler) handleCancel(request *jsonrpc2.Request) (result interface{}, err error) {
	if request.Params == nil {
		return nil, nil
	}

}