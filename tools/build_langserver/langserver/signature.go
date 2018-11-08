package langserver

import (
	"context"
	"encoding/json"
	"tools/build_langserver/lsp"

	"fmt"
	"github.com/sourcegraph/jsonrpc2"
)

const signatureMethod = "textDocument/signatureHelp"

func (h *LsHandler) handleSignature(ctx context.Context, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	log.Info("signature with params %s", req.Params)
	var params lsp.TextDocumentPositionParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	documentURI, err := getURIAndHandleErrors(params.TextDocument.URI, signatureMethod)
	if err != nil {
		return nil, err
	}

	position := params.Position

	h.mu.Lock()
	defer h.mu.Unlock()
	signatures, err := h.getSignatures(ctx, documentURI, position)
	if err != nil {
		return nil, err
	}

	return &lsp.SignatureHelp{
		Signatures:      signatures,
		ActiveSignature: 0,
	}, nil
}

func (h *LsHandler) getSignatures(ctx context.Context, uri lsp.DocumentURI, pos lsp.Position) ([]lsp.SignatureInformation, error) {
	fileContent := h.workspace.documents[uri].textInEdit
	lineContent := h.ensureLineContent(uri, pos)

	log.Info("Signature help lineContent: %s", lineContent)

	var sigList []lsp.SignatureInformation
	//var sigErr error

	if isEmpty(lineContent, pos) {
		return sigList, nil
	}

	lineContent = lineContent[:pos.Character]

	call := h.analyzer.FuncCallFromContentAndPos(JoinLines(fileContent, true), pos)

	fmt.Println(call, lineContent)
	fmt.Println(h.analyzer.BuiltIns[call.Name])
	return sigList, nil
}

//func signatureFromRuleDef(def *RuleDef) lsp.SignatureInformation {
//
//}
