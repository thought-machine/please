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
	// Read file, and get a list of strings from the file
	// TODO: Try using GetLineContent instead
	fileContent, err := ReadFile(ctx, uri)
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

	lineContent := fileContent[position.Line]

	var contentString string

	switch ident.Type {
	case "call":
		identArgs := ident.Action.Call.Arguments
		contentString = getCallContent(ctx, analyzer, identArgs, ident.Name,
			lineContent, position, uri)
	case "property":
		//TODO(bnmetrics)
	case "assign":
		//TODO(bnmetrics)
		// identAssign := ident.Action.Assign
		fmt.Println(ident.IdentStatement.Action.Assign)
	case "augAssign":
		//TODO(bnmetrics)
	default:
		//TODO(bnmetrics): handle cases when ident.Action is nil
	}

	return &lsp.MarkupContent{
		Value: contentString,
		Kind:  lsp.MarkDown, // TODO(bnmetrics): this might be reconsidered
	}, nil
}

func getCallContent(ctx context.Context, analyzer *Analyzer, args []asp.CallArgument,
	identName string, lineContent string, pos lsp.Position, uri lsp.DocumentURI) string {

	// Return empty string if the hovered content is blank
	if isEmpty(lineContent, pos) {
		return ""
	}
	// check if the hovered line is the first line of the function call
	if strings.Contains(lineContent, identName) {
		return contentFromRuleDef(analyzer, identName)
	}

	// Return the content for the BuildLabel if the line content is a buildLabel
	if content, err := contentFromBuildLabel(ctx, analyzer, lineContent, uri); err == nil && content != "" {
		return content
	}

	if content, err := contentFromIdentArgs(ctx, analyzer, args, identName,
		lineContent, pos, uri); err == nil && content != "" {
		return content
	}

	return ""
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

				if identArg.Value.Val.Property != nil {
					if content := contentFromProperty(ctx, analyzer, identArg.Value.Val.Property,
						lineContent, pos, uri); content != "" {
						return content, nil
					}
				}
				if identArg.Value.Val.String != "" {
					return contentFromBuildLabel(ctx, analyzer, lineContent, uri)
				}
				if identArg.Value.Val.List != nil {
					return contentFromList(ctx, analyzer, identArg.Value.Val.List, lineContent, pos, uri)
				}
				if identArg.Value.Val.Tuple != nil {
					return contentFromList(ctx, analyzer, identArg.Value.Val.Tuple, lineContent, pos, uri)
				}

			}
			// Return definition if the hoverred content is a positional argument
		} else if identArg.Name == "" {
			return builtinRule.ArgMap[builtinRule.Arguments[i].Name].definition, nil
		}

		// Check if each identStatement argument are assigned to a call,
		// this applies to both keyword and positional arguments
		if identArg.Value.Val.Ident != nil {
			if content := contentFromIdent(ctx, analyzer, identArg.Value.Val.Ident,
				lineContent, pos, uri); content != "" {
				return content, nil
			}
		}

	}
	return "", nil
}

func contentFromIdent(ctx context.Context, analyzer *Analyzer, IdentValExpr *asp.IdentExpr,
	lineContent string, pos lsp.Position, uri lsp.DocumentURI) string {

	// Check if each identStatement argument are assigned to a call
	withInRange := pos.Line >= IdentValExpr.Pos.Line-1 &&
		pos.Line <= IdentValExpr.EndPos.Line-1

	if withInRange {
		return contentFromIdentExpr(ctx, analyzer, IdentValExpr,
			lineContent, pos, uri)
	}
	return ""
}

func contentFromProperty(ctx context.Context, analyzer *Analyzer, propertyVal *asp.IdentExpr,
	lineContent string, pos lsp.Position, uri lsp.DocumentURI) string {

	withInRange := pos.Character >= propertyVal.Pos.Column-1 &&
		pos.Character <= propertyVal.EndPos.Column-1

	if withInRange {
		return contentFromIdentExpr(ctx, analyzer, propertyVal,
			lineContent, pos, uri)
	}

	return ""
}

func contentFromIdentExpr(ctx context.Context, analyzer *Analyzer, identExpr *asp.IdentExpr,
	lineContent string, pos lsp.Position, uri lsp.DocumentURI) string {

	if identExpr.Action != nil {
		for _, action := range identExpr.Action {
			if action.Call != nil {
				content := getCallContent(ctx, analyzer, action.Call.Arguments,
					identExpr.Name, lineContent, pos, uri)
				if content != "" {
					return content
				}
			}
		}
	}

	return ""
}

func contentFromList(ctx context.Context, analyzer *Analyzer, listVal *asp.List,
	lineContent string, pos lsp.Position, uri lsp.DocumentURI) (string, error) {

	for _, expr := range listVal.Values {
		withInRange := pos.Character >= expr.Pos.Column-1 &&
			pos.Character <= expr.EndPos.Column-1

		if withInRange && expr.Val.String != "" {
			return contentFromBuildLabel(ctx, analyzer, expr.Val.String, uri)
		}

		if expr.Val.List != nil {
			return contentFromList(ctx, analyzer, expr.Val.List, lineContent, pos, uri)
		}
		if expr.Val.Tuple != nil {
			return contentFromList(ctx, analyzer, expr.Val.Tuple, lineContent, pos, uri)
		}
	}

	return "", nil
}

func isEmpty(lineContent string, pos lsp.Position) bool {
	return len(lineContent) < pos.Character+1 || strings.TrimSpace(lineContent[:pos.Character]) == ""
}

func contentFromBuildLabel(ctx context.Context, analyzer *Analyzer,
	lineContent string, uri lsp.DocumentURI) (string, error) {

	trimed := TrimQoutes(lineContent)
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
