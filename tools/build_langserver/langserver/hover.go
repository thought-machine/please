package langserver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sourcegraph/jsonrpc2"
	"tools/build_langserver/lsp"
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
	content, err := getHoverContent(ctx, documentURI, position)
	h.mu.Unlock()

	if err != nil {
		return nil, err
	}

	return &lsp.Hover{
		Contents: content,
	}, nil
}

func getHoverContent(ctx context.Context, uri lsp.DocumentURI, position lsp.Position) (content []lsp.MarkedString, err error) {
	// Read file,
	fileContent, err := ReadFile(ctx, uri)
	if err != nil {
		return nil, &jsonrpc2.Error{
			Code: jsonrpc2.CodeParseError,
			Message: fmt.Sprintf("fail to read file %s", uri),
		}
	}
	// get the character from the line
	fmt.Println(fileContent)
	// look up the character from build_defs, and pull out the documentation
	return nil, nil
}
