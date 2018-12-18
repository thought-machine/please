package langserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/thought-machine/please/src/parse/asp"
	"github.com/thought-machine/please/tools/build_langserver/lsp"

	"github.com/sourcegraph/jsonrpc2"
)

const hoverMethod = "textDocument/hover"

func (h *LsHandler) handleHover(ctx context.Context, req *jsonrpc2.Request) (result interface{}, err error) {
	params, err := h.getParamFromTDPositionReq(req, hoverMethod)
	if err != nil {
		return nil, err
	}
	documentURI := params.TextDocument.URI

	h.mu.Lock()
	defer h.mu.Unlock()
	content, err := h.getHoverContent(ctx, documentURI, params.Position)

	if err != nil {
		return nil, err
	}

	markedString := lsp.MarkedString{
		Language: "build",
		Value:    content,
	}
	markedStrings := []lsp.MarkedString{markedString}

	log.Info("hover content: %s", markedStrings)

	return &lsp.Hover{
		Contents: markedStrings,
		// TODO(bnmetrics): we can add range here later
	}, nil
}

func (h *LsHandler) getHoverContent(ctx context.Context, uri lsp.DocumentURI, pos lsp.Position) (content string, err error) {
	// Get the content of the line from the position
	fileContent := h.workspace.documents[uri].textInEdit
	if pos.Line >= len(fileContent) {
		return "", &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: fmt.Sprintf("Invalid file line: requested %d, but we think file is only %d long", pos.Line, len(fileContent)),
		}
	}
	lineContent := fileContent[pos.Line]

	if err != nil {
		return "", &jsonrpc2.Error{
			Code:    jsonrpc2.CodeParseError,
			Message: fmt.Sprintf("fail to read file %s: %s", uri, err),
		}
	}

	// Return empty string if the hovered content is blank
	if isEmpty(lineContent, pos) {
		return "", nil
	}
	stmts := h.analyzer.AspStatementFromContent(JoinLines(fileContent, true))

	call := h.analyzer.CallFromAST(stmts, pos)
	label := h.analyzer.BuildLabelFromAST(ctx, stmts, uri, pos)
	var contentString string
	var contentErr error

	if call != nil {
		subincludes := h.analyzer.GetSubinclude(ctx, stmts, uri)
		rule := h.analyzer.GetBuildRuleByName(call.Name, subincludes)

		if rule != nil {
			contentString, contentErr = contentFromCall(call.Arguments, rule, lineContent, pos)
		}
	}

	if label != nil {
		if label.BuildDef != nil && label.BuildDef.Content != "" {
			contentString = label.BuildDef.Content
		} else {
			contentString = label.Definition
		}
	}

	if contentErr != nil {
		log.Warning("fail to get content from Build file %s: %s", uri, contentErr)
		return "", nil
	}
	return contentString, nil
}

func contentFromCall(args []asp.CallArgument, ruleDef *RuleDef,
	lineContent string, pos lsp.Position) (string, error) {

	// check if the hovered content is on the name of the ident
	if strings.Contains(lineContent, ruleDef.Name) {
		// adding the trailing open paren to the identName ensure it's a call,
		// prevent inaccuracy for cases like so: replace_str = x.replace('-', '_')
		identNameIndex := strings.Index(lineContent, ruleDef.Name+"(")

		if pos.Character >= identNameIndex &&
			pos.Character <= identNameIndex+len(ruleDef.Name)-1 {
			return contentFromRuleDef(ruleDef), nil
		}
	}

	// Check arguments of the IdentStatement, and return the appropriate content if any
	return contentFromArgs(args, ruleDef, pos)
}

func contentFromArgs(args []asp.CallArgument, ruleDef *RuleDef, pos lsp.Position) (string, error) {

	for i, identArg := range args {
		argNameEndPos := asp.Position{
			Line:   identArg.Pos.Line,
			Column: identArg.Pos.Column + len(identArg.Name),
		}
		if withInRange(identArg.Pos, argNameEndPos, pos) {
			// This is to prevent cases like str.format(),
			// When the positional args are not exactly stored in ArgMap
			arg, okay := ruleDef.ArgMap[identArg.Name]
			if okay {
				return arg.Definition, nil
			}
			// Return definition if the hovered content is a positional argument
		} else if identArg.Name == "" && withInRange(identArg.Value.Pos, identArg.Value.EndPos, pos) {
			argInd := i
			if ruleDef.Arguments[0].Name == "self" {
				argInd++
			}
			return ruleDef.ArgMap[ruleDef.Arguments[argInd].Name].Definition, nil
		}

	}
	return "", nil
}

// contentFromRuleDef returns the content from when hovering over the name of a function call
// return value consist of a string containing the header and the docstring of a build rule
func contentFromRuleDef(ruleDef *RuleDef) string {

	header := ruleDef.Header
	docString := ruleDef.Docstring

	if docString != "" {
		return header + "\n\n" + docString
	}
	return header
}
