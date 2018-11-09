package langserver

import (
	"context"
	"encoding/json"
	"tools/build_langserver/lsp"

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
	builtRule, present := h.analyzer.BuiltIns[call.Name]
	if !present {
		return sigList, nil
	}

	sigInfo := lsp.SignatureInformation{
		Label:         call.Name,
		Documentation: builtRule.Docstring,
		Parameters:    paramsFromRuleDef(builtRule),
	}

	return append(sigList, sigInfo), nil
}

func paramsFromRuleDef(def *RuleDef) []lsp.ParameterInformation {

	var params []lsp.ParameterInformation

	for _, arg := range def.Arguments {
		if !arg.IsPrivate && (arg.Name != "self" && def.Object != "") {
			param := lsp.ParameterInformation{
				Label: arg.Repr,
			}
			params = append(params, param)
		}
	}
	return params
}
