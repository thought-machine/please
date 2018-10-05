package langserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sourcegraph/jsonrpc2"

	"tools/build_langserver/lsp"
	"strings"
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
			Code: jsonrpc2.CodeParseError,
			Message: fmt.Sprintf("fail to read file %s", uri),
		}
	}

	// Get Hover Identifier
	ident, err := analyzer.IdentFromPos(uri, position, fileContent)
	if err != nil {
		return nil, &jsonrpc2.Error{
			Code: jsonrpc2.CodeParseError,
			Message: fmt.Sprintf("fail to parse Build file %s", uri),
		}
	}

	lineContent := fileContent[position.Line]

	var contentString string
	// check if the hovered line is an build target indentifier definition
	if strings.Contains(lineContent, ident.Name) {
		header := analyzer.BuiltIns[ident.Name].Header
		docString := analyzer.BuiltIns[ident.Name].Docstring

		contentString = header + "\n\n" + docString
	}

	// check if the hovered line is an argument to the Identifier
	// TODO: THE FOLLOWING NEEDS TO BE REVAMPED, I CHANGED MY MIND
	if strings.Contains(lineContent, "=") && ident.Action.Call != nil {
		EqualIndex := strings.Index(lineContent, "=")
		if position.Character < EqualIndex {
			arg := strings.TrimSpace(lineContent[:EqualIndex])

			IdentArgs := ident.Action.Call.Arguments

			for _, IdenArg := range IdentArgs {
				// Ensure we are not getting the arguments from nested calls
				if IdenArg.Name == arg {
					contentString = analyzer.BuiltIns[ident.Name].ArgMap[arg].definition
				} else {
					//TODO: get the content from nested called, do something with the ident.Call.Arguments
				}
			}
		} else {

		}
	}

	return &lsp.MarkupContent{
		Value: contentString,
		Kind:lsp.MarkDown, // TODO: this might be reconsidered
	}, nil
}
