package langserver

import (
	"context"
	"encoding/json"
	"github.com/thought-machine/please/tools/build_langserver/lsp"

	"github.com/sourcegraph/jsonrpc2"
)

func (h *LsHandler) handleTDRequests(ctx context.Context, req *jsonrpc2.Request) (result interface{}, err error) {
	if !isTextDocumentMethod(req) {
		return nil, nil
	}

	switch req.Method {
	case "textDocument/didOpen":
		var params lsp.DidOpenTextDocumentParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}

		documentURI, err := getURIAndHandleErrors(params.TextDocument.URI, "textDocument/didOpen")
		if err != nil {
			return nil, err
		}

		h.workspace.Store(documentURI, params.TextDocument.Text, params.TextDocument.Version)
		h.diagPublisher.queue.Put(taskDef{uri: documentURI, content: params.TextDocument.Text})

		return nil, nil
	case "textDocument/didChange":
		var params lsp.DidChangeTextDocumentParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}

		documentURI, err := getURIAndHandleErrors(params.TextDocument.URI, "textDocument/didChange")
		if err != nil {
			return nil, err
		}

		if err := h.workspace.TrackEdit(documentURI, params.ContentChanges, params.TextDocument.Version); err != nil {
			return nil, err
		}

		task := taskDef{
			uri:     documentURI,
			content: JoinLines(h.workspace.documents[documentURI].textInEdit, true),
		}
		h.diagPublisher.queue.Put(task)

		return nil, nil
	case "textDocument/didSave":
		var params lsp.DidSaveTextDocumentParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}
		if params.Text == "" {
			return nil, nil
		}

		documentURI, err := getURIAndHandleErrors(params.TextDocument.URI, "textDocument/didSave")
		if err != nil {
			return nil, err
		}
		return nil, h.workspace.Update(documentURI, params.Text)
	case "textDocument/didClose":
		var params lsp.DidCloseTextDocumentParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return nil, err
		}

		documentURI, err := getURIAndHandleErrors(params.TextDocument.URI, "textDocument/didClose")
		if err != nil {
			return nil, err
		}

		if err := h.workspace.Close(documentURI); err != nil {
			return nil, err
		}

		return nil, nil
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

func isTextDocumentMethod(req *jsonrpc2.Request) bool {
	return req.Method == "textDocument/didOpen" ||
		req.Method == "textDocument/didChange" ||
		req.Method == "textDocument/didClose" ||
		req.Method == "textDocument/didSave" ||
		req.Method == "textDocument/willSave"
}
