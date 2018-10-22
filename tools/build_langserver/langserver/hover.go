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
	documentURI, err := EnsureURL(params.TextDocument.URL, "file")
	if err != nil {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: fmt.Sprintf("invalid documentURI '%s' for method %s", documentURI, hoverMethod),
		}
	}
	position := params.Position

	h.mu.Lock()
	content, err := getHoverContent(ctx, h.analyzer, documentURI, position)
	h.mu.Unlock()

	if err != nil {
		return nil, err
	}

	return &lsp.Hover{
		Contents: *content,
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
	ident, err := analyzer.IdentFromPos(uri, position)
	if err != nil {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeParseError,
			Message: fmt.Sprintf("fail to parse Build file %s", uri),
		}
	}

	// Return empty string if the hovered content is blank
	if isEmpty(lineContent[0], position) {
		return &lsp.MarkupContent{
			Value:"",
			Kind:  lsp.MarkDown,
		}, nil
	}

	var contentString string
	var contentErr error
	switch ident.Type {
	case "call":
		identArgs := ident.Action.Call.Arguments
		contentString, contentErr = getCallContent(ctx, analyzer, identArgs, ident.Name,
								lineContent[0], position, uri)
	case "property":
		//TODO(bnmetrics)
	case "assign":
		contentString, contentErr = contentFromExpression(ctx, analyzer, ident.Action.Assign,
													      lineContent[0], position, uri)
	case "augAssign":
		//TODO(bnmetrics)
	default:
		//TODO(bnmetrics): handle cases when ident.Action is nil
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

func getCallContent(ctx context.Context, analyzer *Analyzer, args []asp.CallArgument,
	identName string, lineContent string, pos lsp.Position, uri lsp.DocumentURI) (string, error) {

	// check if the hovered line is the first line of the function call
	if strings.Contains(lineContent, identName) {
		return contentFromRuleDef(analyzer, identName), nil
	}

	// Return the content for the BuildLabel if the line content is a buildLabel
	content, err := contentFromBuildLabel(ctx, analyzer, lineContent, uri)
	if err != nil {
		return "", err
	}
	if content != "" {
		return content, nil
	}

	// Check arguments of the IdentStatement, and return the appropriate content if any
	return contentFromIdentArgs(ctx, analyzer, args, identName,
		 								lineContent, pos, uri)
}

func contentFromExpression(ctx context.Context, analyzer *Analyzer, expr *asp.Expression,
	lineContent string, pos lsp.Position, uri lsp.DocumentURI) (string, error) {

	withInLineRange := pos.Line >= expr.Pos.Line-1 &&
					   pos.Line <= expr.EndPos.Line-1

	withInColRange := pos.Character >= expr.Pos.Column-1 &&
					   pos.Character <= expr.EndPos.Column-1

	onTheSameLine := pos.Line == expr.EndPos.Line - 1 &&
		              pos.Line == expr.Pos.Line - 1

	if !withInLineRange || (onTheSameLine && !withInColRange) {
		return "", nil
	}

	// content from Expression.If
	if expr.If != nil {
		if expr.If.Condition != nil {
			content, err := contentFromExpression(ctx, analyzer, expr.If.Condition, lineContent, pos, uri)
			if err != nil {
				return "", err
			}
			if content != "" {
				return content, nil
			}
		}
		if expr.If.Else != nil {
			content, err := contentFromExpression(ctx, analyzer, expr.If.Else, lineContent, pos, uri)
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
		return  contentFromValueExpression(ctx, analyzer, expr.Val, lineContent, pos, uri)
	}

	// content from Expression.UnaryOp
	if expr.UnaryOp != nil && &expr.UnaryOp.Expr != nil {
		return contentFromValueExpression(ctx, analyzer, &expr.UnaryOp.Expr,
										  lineContent, pos, uri)
	}

	return "", nil
}

func contentFromIdentArgs(ctx context.Context, analyzer *Analyzer, args []asp.CallArgument,
	identName string, lineContent string, pos lsp.Position, uri lsp.DocumentURI) (string, error) {

	builtinRule := analyzer.BuiltIns[identName]
	for i, identArg := range args {

		// check if the lineContent contains "=", as it could be a keyword argument
		if strings.Contains(lineContent, "=") {

			EqualIndex := strings.Index(lineContent, "=")
			hoveredArgName := strings.TrimSpace(lineContent[:EqualIndex])

			if identArg.Name == hoveredArgName {
				// Return the definition of the argument if the hovering is on the argument name
				if pos.Character <= EqualIndex {
					return analyzer.BuiltIns[identName].ArgMap[hoveredArgName].definition, nil
				}

				return contentFromValueExpression(ctx, analyzer, identArg.Value.Val,
												  lineContent, pos, uri)
			}
		// Return definition if the hovered content is a positional argument
		} else if identArg.Name == "" {
			return builtinRule.ArgMap[builtinRule.Arguments[i].Name].definition, nil
		}

		// Check if each identStatement argument are assigned to a call,
		// this applies to both keyword and positional arguments
		if identArg.Value.Val.Ident != nil {
			content, err := contentFromIdent(ctx, analyzer, identArg.Value.Val.Ident,
				lineContent, pos, uri)
			if err != nil {
				return "", err
			}
			if content != "" {
				return content, nil
			}
		}

	}
	return "", nil
}

func contentFromValueExpression(ctx context.Context, analyzer *Analyzer,
	valExpr *asp.ValueExpression, lineContent string, pos lsp.Position, uri lsp.DocumentURI) (string, error) {
	if valExpr.Property != nil {
		content, err := contentFromProperty(ctx, analyzer, valExpr.Property,
			lineContent, pos, uri)
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
		return contentFromList(ctx, analyzer, valExpr.List, lineContent, pos, uri)
	}
	if valExpr.Tuple != nil {
		return contentFromList(ctx, analyzer, valExpr.Tuple, lineContent, pos, uri)
	}
	if valExpr.Ident != nil {
		return contentFromIdent(ctx, analyzer, valExpr.Ident,
								lineContent, pos, uri)
	}

	return "", nil
}

// contentFromIdent returns hover content from ValueExpression.Ident
func contentFromIdent(ctx context.Context, analyzer *Analyzer, IdentValExpr *asp.IdentExpr,
	lineContent string, pos lsp.Position, uri lsp.DocumentURI) (string, error) {

	if pos.Line >= IdentValExpr.Pos.Line-1 &&
		pos.Line <= IdentValExpr.EndPos.Line-1 {

		return contentFromIdentExpr(ctx, analyzer, IdentValExpr,
			lineContent, pos, uri)
	}
	return "", nil
}

// contentFromProperty returns hover content from ValueExpression.Property
func contentFromProperty(ctx context.Context, analyzer *Analyzer, propertyVal *asp.IdentExpr,
	lineContent string, pos lsp.Position, uri lsp.DocumentURI) (string, error)  {

	if pos.Character >= propertyVal.Pos.Column-1 &&
		pos.Character <= propertyVal.EndPos.Column-1 {

		return contentFromIdentExpr(ctx, analyzer, propertyVal,
			lineContent, pos, uri)
	}

	return "", nil
}

func contentFromIdentExpr(ctx context.Context, analyzer *Analyzer, identExpr *asp.IdentExpr,
	lineContent string, pos lsp.Position, uri lsp.DocumentURI) (string, error) {

	if identExpr.Action != nil {
		for _, action := range identExpr.Action {
			if action.Call != nil {
				return getCallContent(ctx, analyzer, action.Call.Arguments,
					identExpr.Name, lineContent, pos, uri)
			}
			if action.Property != nil {
				return contentFromProperty(ctx, analyzer, action.Property,
					lineContent, pos, uri)
			}
		}
	}

	return "", nil
}

func contentFromList(ctx context.Context, analyzer *Analyzer, listVal *asp.List,
	lineContent string, pos lsp.Position, uri lsp.DocumentURI) (string, error) {

	for _, expr := range listVal.Values {
		withInRange := pos.Character >= expr.Pos.Column-1 &&
					   pos.Character <= expr.EndPos.Column-1

		onTheSameLine := pos.Line == expr.EndPos.Line - 1 &&
						 pos.Line == expr.Pos.Line - 1

		if onTheSameLine && withInRange && expr.Val.String != "" {
			return contentFromBuildLabel(ctx, analyzer, expr.Val.String, uri)
		}

		content, err := contentFromValueExpression(ctx, analyzer, expr.Val,
												   lineContent, pos, uri)
		if err != nil {
			return "", err
		}
		if content != "" {
			return content, nil
		}
	}

	return "", nil
}

func isEmpty(lineContent string, pos lsp.Position) bool {
	return len(lineContent) < pos.Character + 1 || strings.TrimSpace(lineContent[:pos.Character]) == ""
}

func contentFromBuildLabel(ctx context.Context, analyzer *Analyzer,
	lineContent string, uri lsp.DocumentURI) (string, error) {

	trimed := TrimQuotes(lineContent)
	if core.LooksLikeABuildLabel(trimed) {
		buildLabel, err := analyzer.BuildLabelFromString(ctx, core.RepoRoot, uri, trimed)
		if err != nil {
			return "", err
		}
		return buildLabel.BuildDefContent, nil
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
