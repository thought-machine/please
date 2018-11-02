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

	log.Info("Hover with param %s", req.Params)
	documentURI, err := EnsureURL(params.TextDocument.URI, "file")
	if err != nil {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: fmt.Sprintf("invalid documentURI '%s' for method %s", documentURI, hoverMethod),
		}
	}
	if !h.analyzer.IsBuildFile(documentURI) {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: fmt.Sprintf("documentURI '%s' is not supported because it's not a buildfile", documentURI),
		}
	}

	position := params.Position

	h.mu.Lock()
	content, err := getHoverContent(ctx, h.analyzer, documentURI, position)
	h.mu.Unlock()

	if err != nil {
		return nil, err
	}

	log.Info("hover content: %s", content)
	// TODO(bnm): reconsider the content, because right now everything is on one line.....:(
	return &lsp.Hover{
		Contents: *content,
		// TODO(bnmetrics): we can add range here later
	}, nil
}

func getHoverContent(ctx context.Context, analyzer *Analyzer,
	uri lsp.DocumentURI, position lsp.Position) (content *lsp.MarkupContent, err error) {
	// Get the content of the line from the position
	lineContent, err := GetLineContent(ctx, uri, position)
	if err != nil {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeParseError,
			Message: fmt.Sprintf("fail to read file %s", uri),
		}
	}

	// Get Hover Identifier
	stmt, err := analyzer.StatementFromPos(uri, position)
	if err != nil {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeParseError,
			Message: fmt.Sprintf("fail to parse Build file %s", uri),
		}
	}

	emptyContent := &lsp.MarkupContent{
		Value: "",
		Kind:  lsp.MarkDown,
	}
	// Return empty string if the hovered content is blank
	if isEmpty(lineContent[0], position) || stmt == nil {
		return emptyContent, nil
	}

	var contentString string
	var contentErr error
	if stmt.Ident != nil {
		ident := stmt.Ident
		switch ident.Type {
		case "call":
			identArgs := ident.Action.Call.Arguments
			contentString, contentErr = contentFromCall(ctx, analyzer, identArgs, ident.Name,
				lineContent[0], uri, position)
		case "property":
			contentString, contentErr = contentFromProperty(ctx, analyzer, ident.Action.Property,
				lineContent[0], uri, position)
		case "assign":
			contentString, contentErr = contentFromExpression(ctx, analyzer, ident.Action.Assign,
				lineContent[0], uri, position)
		case "augAssign":
			contentString, contentErr = contentFromExpression(ctx, analyzer, ident.Action.AugAssign,
				lineContent[0], uri, position)
		default:
			return emptyContent, nil
		}
	} else if stmt.Expression != nil {
		contentString, contentErr = contentFromExpression(ctx, analyzer, stmt.Expression,
			lineContent[0], uri, position)
	}

	if contentErr != nil {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeParseError,
			Message: fmt.Sprintf("fail to get content from Build file %s", uri),
		}
	}
	return &lsp.MarkupContent{
		Value: contentString,
		Kind:  lsp.MarkDown, // TODO(bnmetrics): this might be reconsidered
	}, nil
}

func contentFromCall(ctx context.Context, analyzer *Analyzer, args []asp.CallArgument,
	identName string, lineContent string, uri lsp.DocumentURI, pos lsp.Position) (string, error) {

	// check if the hovered content is on the name of the ident
	if strings.Contains(lineContent, identName) {
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
				return arg.definition, nil
			}
			// Return definition if the hovered content is a positional argument
		} else if identArg.Name == "" && withInRange(identArg.Value.Pos, identArg.Value.EndPos, pos) {
			argInd := i
			if builtinRule.Arguments[0].Name == "self" {
				argInd++
			}
			return builtinRule.ArgMap[builtinRule.Arguments[argInd].Name].definition, nil
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

func contentFromValueExpression(ctx context.Context, analyzer *Analyzer,
	valExpr *asp.ValueExpression, lineContent string, uri lsp.DocumentURI, pos lsp.Position) (string, error) {
	if valExpr.Property != nil {
		content, err := contentFromProperty(ctx, analyzer, valExpr.Property,
			lineContent, uri, pos)
		if err != nil {
			return "", err
		}
		if content != "" {
			return content, nil
		}
	}
	if valExpr.String != "" && strings.Contains(lineContent, valExpr.String) {
		return contentFromBuildLabel(ctx, analyzer, lineContent, uri)
	}
	if valExpr.List != nil {
		return contentFromList(ctx, analyzer, valExpr.List, lineContent, uri, pos)
	}
	if valExpr.Tuple != nil {
		return contentFromList(ctx, analyzer, valExpr.Tuple, lineContent, uri, pos)
	}
	if valExpr.Ident != nil {
		return contentFromIdent(ctx, analyzer, valExpr.Ident,
			lineContent, uri, pos)
	}

	return "", nil
}

// contentFromIdent returns hover content from ValueExpression.Ident
func contentFromIdent(ctx context.Context, analyzer *Analyzer, identValExpr *asp.IdentExpr,
	lineContent string, uri lsp.DocumentURI, pos lsp.Position) (string, error) {

	if withInRange(identValExpr.Pos, identValExpr.EndPos, pos) {
		return contentFromIdentExpr(ctx, analyzer, identValExpr,
			lineContent, uri, pos)
	}

	return "", nil
}

// contentFromProperty returns hover content from ValueExpression.Property
func contentFromProperty(ctx context.Context, analyzer *Analyzer, propertyVal *asp.IdentExpr,
	lineContent string, uri lsp.DocumentURI, pos lsp.Position) (string, error) {

	if withInRange(propertyVal.Pos, propertyVal.EndPos, pos) {

		return contentFromIdentExpr(ctx, analyzer, propertyVal,
			lineContent, uri, pos)
	}

	return "", nil
}

func contentFromIdentExpr(ctx context.Context, analyzer *Analyzer, identExpr *asp.IdentExpr,
	lineContent string, uri lsp.DocumentURI, pos lsp.Position) (string, error) {

	if identExpr.Action != nil {
		for _, action := range identExpr.Action {
			if action.Call != nil {
				return contentFromCall(ctx, analyzer, action.Call.Arguments,
					identExpr.Name, lineContent, uri, pos)
			}
			if action.Property != nil {
				return contentFromProperty(ctx, analyzer, action.Property,
					lineContent, uri, pos)
			}
		}
	}

	return "", nil
}

func contentFromList(ctx context.Context, analyzer *Analyzer, listVal *asp.List,
	lineContent string, uri lsp.DocumentURI, pos lsp.Position) (string, error) {

	for _, expr := range listVal.Values {
		if withInRange(expr.Pos, expr.EndPos, pos) && expr.Val.String != "" {
			return contentFromBuildLabel(ctx, analyzer, expr.Val.String, uri)
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
	lineContent string, uri lsp.DocumentURI) (string, error) {

	trimed := TrimQuotes(lineContent)

	if core.LooksLikeABuildLabel(trimed) {
		buildLabel, err := analyzer.BuildLabelFromString(ctx, uri, trimed)
		if err != nil {
			return "", err
		}
		return buildLabel.Definition, nil
	}

	return "", nil
}

// contentFromRuleDef returns a string contains the header and the docstring of a build rule
func contentFromRuleDef(analyzer *Analyzer, name string) string {

	header := analyzer.BuiltIns[name].Header
	docString := analyzer.BuiltIns[name].Docstring

	if docString != "" {
		return header + "\n\n" + docString
	}
	return header
}

// withInRange checks if the input position from lsp is within the range of the Expression
func withInRange(exprPos asp.Position, exprEndPos asp.Position, pos lsp.Position) bool {
	withInLineRange := pos.Line >= exprPos.Line-1 &&
		pos.Line <= exprEndPos.Line-1

	withInColRange := pos.Character >= exprPos.Column-1 &&
		pos.Character <= exprEndPos.Column-1

	onTheSameLine := pos.Line == exprEndPos.Line-1 &&
		pos.Line == exprPos.Line-1

	if !withInLineRange || (onTheSameLine && !withInColRange) {
		return false
	}

	return true
}
