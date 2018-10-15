package langserver

import (
	"context"
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

func getHoverContent(ctx context.Context, analyzer *Analyzer, uri lsp.DocumentURI, position lsp.Position) (content *lsp.MarkupContent, err error) {
	// Read file, and get a list of strings from the file
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
		contentString = getCallContent(lineContent, ident, analyzer, position)
	case "property":
		//TODO
	case "assign":
		//TODO
	case "augAssign":
		//TODO
	default:
		//TODO(bnmetrics): handle cases when ident.Action is nil
	}

	return &lsp.MarkupContent{
		Value: contentString,
		Kind:  lsp.MarkDown, // TODO(bnmetrics): this might be reconsidered
	}, nil
}

func getCallContent(lineContent string, ident *Identifier, analyzer *Analyzer, position lsp.Position) string {
	var contentString string

	// check if the hovered line is the first line of the function call
	if strings.Contains(lineContent, ident.Name) {
		return getRuleDefContent(analyzer, ident.Name)
	}

	identArgs := ident.Action.Call.Arguments
	for _, identArg := range identArgs {
		if content := ContentFromNestedCall(analyzer, identArg, lineContent, position); content != "" {
			return content
		}

		if content := getArgContent(analyzer, identArg, ident.Name, lineContent, position); content != "" {
			return content
		}

	}

	//// check if the hovered line is an argument to the Identifier
	//// TODO(bnmetrics): revamp!!
	//if strings.Contains(lineContent, "=") {
	//	EqualIndex := strings.Index(lineContent, "=")
	//	// Check if the hover column is the argument definition
	//	if position.Character < EqualIndex {
	//
	//		arg := strings.TrimSpace(lineContent[:EqualIndex])
	//		identArgs := ident.Action.Call.Arguments
	//
	//		for _, identArg := range identArgs {
	//			// Ensure we are not getting the arguments from nested calls
	//			if identArg.Name == arg {
	//				contentString = analyzer.BuiltIns[ident.Name].ArgMap[arg].definition
	//				break
	//			} else {
	//				//TODO(bnmetrics): get the content from nested called, do something with the ident.Call.Arguments
	//				contentString = ContentFromNestedCall(analyzer, identArg, position)
	//
	//				//nestedIdent := identArg.Value.Val.Ident
	//				//withInRange := bool(position.Line >= identArg.Value.Pos.Line - 1 && position.Line <= (identArg.Value.EndPos.Line - 1))
	//				//if nestedIdent != nil {
	//				//	fmt.Println(lineContent, nestedIdent.Name, position.Line, identArg.Value.EndPos.Line - 1)
	//				//}
	//				//if nestedIdent != nil && withInRange && nestedIdent.Action != nil {
	//				//	contentstring := analyzer.BuiltIns[nestedIdent.Name].ArgMap
	//				//	fmt.Println(contentstring)
	//				//}
	//			}
	//		}
	//	} else {
	//
	//	}
	//}

	return contentString
}

func ContentFromNestedCall(analyzer *Analyzer, identArg asp.CallArgument, lineContent string, position lsp.Position) string {
	nestedIdent := identArg.Value.Val.Ident
	withInRange := bool(position.Line >= identArg.Value.Pos.Line-1 &&
		position.Line <= identArg.Value.EndPos.Line-1)

	if nestedIdent != nil && withInRange && nestedIdent.Action != nil {
		for _, action := range nestedIdent.Action {
			if action.Call != nil {
				for _, arg := range action.Call.Arguments {
					if content := getArgContent(analyzer, arg, nestedIdent.Name, lineContent, position); content != "" {
						return content
					}
				}
			}
		}
		content := getRuleDefContent(analyzer, nestedIdent.Name)
		return content
	}

	return ""
}

func getRuleDefContent(analyzer *Analyzer, name string) string {
	header := analyzer.BuiltIns[name].Header
	docString := analyzer.BuiltIns[name].Docstring

	return header + "\n\n" + docString
}

func getArgContent(analyzer *Analyzer, identArg asp.CallArgument, name string, lineContent string, position lsp.Position) string {
	if strings.Contains(lineContent, "=") {
		EqualIndex := strings.Index(lineContent, "=")
		if position.Character < EqualIndex {
			arg := strings.TrimSpace(lineContent[:EqualIndex])

			if identArg.Name == arg {
				return analyzer.BuiltIns[name].ArgMap[arg].definition
			}
		}
	}

	return ""
}
