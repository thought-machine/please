package langserver

import (
	"context"
	"encoding/json"

	"tools/build_langserver/lsp"

	"github.com/sourcegraph/jsonrpc2"
)

func (h *LsHandler) handleFSRequests(ctx context.Context, req *jsonrpc2.Request) (result interface{}, err error) {
	if !isFileSystemMethod(req) {
		return nil, nil
	}

	log.Info("Handling fs method %s, with param %s", req.Method, req.Params)

	switch req.Method {
	case "textDocument/didOpen":
		var params lsp.DidOpenTextDocumentParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}
		h.workspace.Store(params.TextDocument.URI, params.TextDocument.Text)
		return nil, nil
	case "textDocument/didChange":
		var params lsp.DidChangeTextDocumentParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}
		return nil, h.workspace.TrackEdit(params.TextDocument.URI, params.ContentChanges)
	case "textDocument/didSave":
		var params lsp.DidSaveTextDocumentParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}
		return nil, h.workspace.Update(params.TextDocument.URI, params.Text)
	case "textDocument/didClose":
		var params lsp.DidCloseTextDocumentParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}
		return nil, h.workspace.Close(params.TextDocument.URI)
	case "textDocument/willSave":
		var params lsp.WillSaveTextDocumentParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}
		// no-op
		return nil, nil
	default:
		log.Error("unexpected file system request method: %s", req.Method)
		return nil, nil
	}
}

func isFileSystemMethod(req *jsonrpc2.Request) bool {
	return req.Method == "textDocument/didOpen" ||
		req.Method == "textDocument/didChange" ||
		req.Method == "textDocument/didClose" ||
		req.Method == "textDocument/didSave" ||
		req.Method == "textDocument/willSave"
}
