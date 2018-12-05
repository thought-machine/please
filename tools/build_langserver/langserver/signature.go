package langserver

import (
	"context"
	"strings"

	"github.com/thought-machine/please/tools/build_langserver/lsp"

	"github.com/sourcegraph/jsonrpc2"
)

const signatureMethod = "textDocument/signatureHelp"

func (h *LsHandler) handleSignature(ctx context.Context, req *jsonrpc2.Request) (result interface{}, err error) {
	params, err := h.getParamFromTDPositionReq(req, signatureMethod)
	if err != nil {
		return nil, err
	}
	documentURI := params.TextDocument.URI

	h.mu.Lock()
	defer h.mu.Unlock()
	signatures := h.getSignatures(ctx, documentURI, params.Position)
	log.Info("signatures, %s", signatures)

	return signatures, nil
}

func (h *LsHandler) getSignatures(ctx context.Context, uri lsp.DocumentURI, pos lsp.Position) *lsp.SignatureHelp {
	fileContent := h.workspace.documents[uri].textInEdit
	lineContent := h.ensureLineContent(uri, pos)

	log.Info("Signature help lineContent: %s", lineContent)

	if isEmpty(lineContent, pos) {
		return nil
	}

	lineContent = lineContent[:pos.Character]

	stmts := h.analyzer.AspStatementFromContent(JoinLines(fileContent, true))

	call := h.analyzer.CallFromAST(stmts, pos)
	if call == nil {
		return nil
	}

	subincludes := h.analyzer.GetSubinclude(ctx, stmts, uri)
	builtRule := h.analyzer.GetBuildRuleByName(call.Name, subincludes)

	if builtRule == nil {
		log.Info("rule %s not present, exit", call.Name)
		return nil
	}
	label := builtRule.Header[strings.Index(builtRule.Header, builtRule.Name)+len(builtRule.Name):]

	sigInfo := lsp.SignatureInformation{
		Label:         label,
		Documentation: builtRule.Docstring,
		Parameters:    paramsFromRuleDef(builtRule),
	}
	return &lsp.SignatureHelp{
		Signatures:      []lsp.SignatureInformation{sigInfo},
		ActiveSignature: 0,
		ActiveParameter: getActiveParam(call, builtRule),
	}
}

func paramsFromRuleDef(def *RuleDef) []lsp.ParameterInformation {

	var params []lsp.ParameterInformation

	for _, arg := range def.Arguments {
		if !arg.IsPrivate && (arg.Name != "self" || def.Object == "") {
			param := lsp.ParameterInformation{
				Label: def.ArgMap[arg.Name].Repr,
			}
			params = append(params, param)
		}
	}
	return params
}

func getActiveParam(callIdent *Call, def *RuleDef) int {
	callArgs := callIdent.Arguments
	if len(callArgs) == 0 {
		return 0
	}

	for i, defArg := range def.Arguments {
		if callArgs[len(callArgs)-1].Name == defArg.Name {
			return i + 1
		}
	}
	return len(callArgs)
}
