package langserver

import (
	"context"
	"encoding/json"
	"fmt"

	"io/ioutil"

	"tools/build_langserver/lsp"

	"github.com/bazelbuild/buildtools/build"
	"github.com/sourcegraph/jsonrpc2"
)

const reformatMethod = "textDocument/formatting"

func (h *LsHandler) handleReformatting(ctx context.Context, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params lsp.DocumentFormattingParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	documentURI, err := getURIAndHandleErrors(params.TextDocument.URI, reformatMethod)
	if err != nil {
		return nil, err
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	edits, err := h.getFormatEdits(documentURI)
	if err != nil {
		log.Warning("error occurred when formatting file: %s, skipping", err)
	}

	return edits, err
}

func (h *LsHandler) getFormatEdits(uri lsp.DocumentURI) ([]*lsp.TextEdit, error) {

	filePath, err := GetPathFromURL(uri, "file")
	if err != nil {
		return nil, err
	}

	bytecontent, err := ioutil.ReadFile(filePath)
	f, err := build.ParseBuild(filePath, bytecontent)

	if err != nil {
		return nil, err
	}

	reformatted := build.Format(f)

	fmt.Println(f)
	fmt.Println(string(reformatted))

	// TODO(bnm): make them into edits
	return nil, nil
}
