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

}

type LsHandler struct {
	init *lsp.InitializeParams
	mu sync.Mutex
}

func (h *LsHandler) Handle (ctx context.Context, con *jsonrpc2.Conn, request *jsonrpc2.Request) {
	// TODO: Select different methods based on request.Method


}

func (h *LsHandler) HandleInit(request jsonrpc2.Request) (result interface{}, err error) {
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

	if params.RootPath != "" {
		params.RootPath = string(params.Root())
	}

	//
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

