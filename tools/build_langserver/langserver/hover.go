package langserver

import (
	"context"
	"fmt"
	"strings"

	"parse/asp"
	"tools/build_langserver/lsp"

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

	call := h.analyzer.FuncCallFromContentAndPos(JoinLines(fileContent, true), pos)
	label := h.analyzer.BuildLabelFromContent(ctx, JoinLines(fileContent, true),
		uri, pos)
	var contentString string
	var contentErr error

	if call != nil {
		contentString, contentErr = contentFromCall(ctx, h.analyzer, call.Arguments, call.Name,
			lineContent, uri, pos)
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

func contentFromCall(ctx context.Context, analyzer *Analyzer, args []asp.CallArgument,
	identName string, lineContent string, uri lsp.DocumentURI, pos lsp.Position) (string, error) {

	// check if the hovered content is on the name of the ident
	if strings.Contains(lineContent, identName) {
		// adding the trailing open paren to the identName ensure it's a call,
		// prevent inaccuracy for cases like so: replace_str = x.replace('-', '_')
		identNameIndex := strings.Index(lineContent, identName+"(")

		if pos.Character >= identNameIndex &&
			pos.Character <= identNameIndex+len(identName)-1 {
			return contentFromRuleDef(analyzer, identName), nil
		}
	}

	// Check arguments of the IdentStatement, and return the appropriate content if any
	return contentFromArgs(analyzer, args, identName, pos)
}

func contentFromArgs(analyzer *Analyzer, args []asp.CallArgument, identName string, pos lsp.Position) (string, error) {

	builtinRule := analyzer.BuiltIns[identName]
	for i, identArg := range args {
		argNameEndPos := asp.Position{
			Line:   identArg.Pos.Line,
			Column: identArg.Pos.Column + len(identArg.Name),
		}
		if withInRange(identArg.Pos, argNameEndPos, pos) {
			// This is to prevent cases like str.format(),
			// When the positional args are not exactly stored in ArgMap
			arg, okay := analyzer.BuiltIns[identName].ArgMap[identArg.Name]
			if okay {
				return arg.Definition, nil
			}
			// Return definition if the hovered content is a positional argument
		} else if identArg.Name == "" && withInRange(identArg.Value.Pos, identArg.Value.EndPos, pos) {
			argInd := i
			if builtinRule.Arguments[0].Name == "self" {
				argInd++
			}
			return builtinRule.ArgMap[builtinRule.Arguments[argInd].Name].Definition, nil
		}

	}
	return "", nil
}

// contentFromRuleDef returns the content from when hovering over the name of a function call
// return value consist of a string containing the header and the docstring of a build rule
func contentFromRuleDef(analyzer *Analyzer, name string) string {

	header := analyzer.BuiltIns[name].Header
	docString := analyzer.BuiltIns[name].Docstring

	if docString != "" {
		return header + "\n\n" + docString
	}
	return header
}
