package langserver

import (
	"context"
	"core"
	"encoding/json"
	"fmt"
	"strings"

	"parse/asp"
	"tools/build_langserver/lsp"

	"github.com/sourcegraph/jsonrpc2"
)

const hoverMethod = "textDocument/hover"

func (h *LsHandler) handleHover(ctx context.Context, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params lsp.TextDocumentPositionParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	documentURI, err := getURIAndHandleErrors(params.TextDocument.URI, hoverMethod)
	if err != nil {
		return nil, err
	}

	position := params.Position

	h.mu.Lock()
	defer h.mu.Unlock()
	content, err := h.getHoverContent(ctx, documentURI, position)

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
	lineContent := h.workspace.documents[uri].textInEdit[pos.Line]

	if err != nil {
		return "", &jsonrpc2.Error{
			Code:    jsonrpc2.CodeParseError,
			Message: fmt.Sprintf("fail to read file %s: %s", uri, err),
		}
	}

	// Get Hover Identifier
	stmt, err := h.analyzer.StatementFromPos(uri, pos)
	if err != nil {
		return "", &jsonrpc2.Error{
			Code:    jsonrpc2.CodeParseError,
			Message: fmt.Sprintf("fail to parse Build file %s", uri),
		}
	}

	// Return empty string if the hovered content is blank
	if isEmpty(lineContent, pos) || stmt == nil {
		return "", nil
	}

	call := h.analyzer.CallFromStatementAndPos(stmt, pos)
	var contentString string
	var contentErr error

	if call != nil {
		contentString, contentErr = contentFromCall(ctx, h.analyzer, call.Arguments, call.Name,
			lineContent, uri, pos)
	} else if stmt.Expression != nil {
		contentString, contentErr = contentFromExpression(ctx, h.analyzer, stmt.Expression,
			lineContent, uri, pos)
	} else if stmt.Ident != nil {
		if stmt.Ident.Type == "assign" {
			contentString, contentErr = contentFromExpression(ctx, h.analyzer, stmt.Ident.Action.Assign,
				lineContent, uri, pos)
		} else if stmt.Ident.Type == "augAssign" {
			contentString, contentErr = contentFromExpression(ctx, h.analyzer, stmt.Ident.Action.AugAssign,
				lineContent, uri, pos)
		} else {
			return "", nil
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
	return contentFromIdentArgs(ctx, analyzer, args, identName,
		lineContent, uri, pos)
}

func contentFromIdentArgs(ctx context.Context, analyzer *Analyzer, args []asp.CallArgument,
	identName string, lineContent string, uri lsp.DocumentURI, pos lsp.Position) (string, error) {

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

		// Get content from the argument value
		content, err := contentFromValueExpression(ctx, analyzer, identArg.Value.Val,
			lineContent, uri, pos)
		if err != nil {
			return "", err
		}
		if content != "" {
			return content, nil
		}

	}
	return "", nil
}

func contentFromExpression(ctx context.Context, analyzer *Analyzer, expr *asp.Expression,
	lineContent string, uri lsp.DocumentURI, pos lsp.Position) (string, error) {

	if !withInRange(expr.Pos, expr.EndPos, pos) {
		return "", nil
	}

	// content from Expression.If
	if expr.If != nil {
		if expr.If.Condition != nil {
			content, err := contentFromExpression(ctx, analyzer, expr.If.Condition, lineContent, uri, pos)
			if err != nil {
				return "", err
			}
			if content != "" {
				return content, nil
			}
		}
		if expr.If.Else != nil {
			content, err := contentFromExpression(ctx, analyzer, expr.If.Else, lineContent, uri, pos)
			if err != nil {
				return "", err
			}
			if content != "" {
				return content, nil
			}
		}
	}
	// content from Expression.Val
	if expr.Val != nil {
		return contentFromValueExpression(ctx, analyzer, expr.Val, lineContent, uri, pos)
	}

	// content from Expression.UnaryOp
	if expr.UnaryOp != nil && &expr.UnaryOp.Expr != nil {
		return contentFromValueExpression(ctx, analyzer, &expr.UnaryOp.Expr,
			lineContent, uri, pos)
	}

	return "", nil
}

func contentFromValueExpression(ctx context.Context, analyzer *Analyzer,
	valExpr *asp.ValueExpression, lineContent string, uri lsp.DocumentURI, pos lsp.Position) (string, error) {

	if valExpr.String != "" && strings.Contains(lineContent, valExpr.String) {
		return contentFromBuildLabel(ctx, analyzer, valExpr.String, uri)
	}
	if valExpr.List != nil {
		return contentFromList(ctx, analyzer, valExpr.List, lineContent, uri, pos)
	}
	if valExpr.Tuple != nil {
		return contentFromList(ctx, analyzer, valExpr.Tuple, lineContent, uri, pos)
	}

	return "", nil
}

func contentFromList(ctx context.Context, analyzer *Analyzer, listVal *asp.List,
	lineContent string, uri lsp.DocumentURI, pos lsp.Position) (string, error) {

	for _, expr := range listVal.Values {
		if !withInRange(expr.Pos, expr.EndPos, pos) {
			continue
		}

		content, err := contentFromValueExpression(ctx, analyzer, expr.Val,
			lineContent, uri, pos)
		if err != nil {
			return "", err
		}
		if content != "" {
			return content, nil
		}
	}

	return "", nil
}

func contentFromBuildLabel(ctx context.Context, analyzer *Analyzer,
	buildLabelstr string, uri lsp.DocumentURI) (string, error) {

	trimed := TrimQuotes(buildLabelstr)

	if core.LooksLikeABuildLabel(trimed) {
		buildLabel, err := analyzer.BuildLabelFromString(ctx, uri, trimed)
		if err != nil {
			return "", err
		}

		if buildLabel.BuildDef != nil && buildLabel.BuildDef.Content != "" {
			return buildLabel.BuildDef.Content, nil
		}
		return buildLabel.Definition, nil
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
