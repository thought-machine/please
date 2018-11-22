package langserver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"tools/build_langserver/lsp"

	"github.com/bazelbuild/buildtools/build"
	"github.com/pmezard/go-difflib/difflib"
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

	doc, ok := h.workspace.documents[uri]
	if !ok {
		return nil, fmt.Errorf("document not opened: %s", uri)
	}

	content := JoinLines(doc.textInEdit, true)
	bytecontent := []byte(content)

	var f *build.File
	if h.analyzer.IsBuildFile(uri) {
		f, err = build.ParseBuild(filePath, bytecontent)
		if err != nil {
			return nil, err
		}
	} else {
		f, err = build.ParseDefault(filePath, bytecontent)
		if err != nil {
			return nil, err
		}
	}

	reformatted := build.Format(f)

	return getEdits(content, string(reformatted)), nil
}

func getEdits(before string, after string) []*lsp.TextEdit {
	beforeLines := difflib.SplitLines(before)
	afterLines := difflib.SplitLines(after)

	matcher := difflib.NewMatcher(beforeLines, afterLines)

	var edits []*lsp.TextEdit
	for _, op := range matcher.GetOpCodes() {
		// Do nothing if it's "e"(equal)
		if op.Tag == 'e' {
			continue
		}

		edit := &lsp.TextEdit{
			Range: lsp.Range{
				Start: lsp.Position{
					Line:      op.I1,
					Character: 0,
				},
				End: lsp.Position{
					Line:      op.I2,
					Character: 0,
				},
			},
		}

		// 'r' means replace, 'i' means insert
		if op.Tag == 'r' || op.Tag == 'i' {
			// since both replaces and inserts are line based,
			// so we add a "\n" at the end of each line if there isn't one
			text := JoinLines(afterLines[op.J1:op.J2], true)
			if strings.HasSuffix(text, "\n") {
				edit.NewText = text
			} else {
				edit.NewText = text + "\n"
			}
		} else if op.Tag == 'd' { // 'd' means delete
			edit.NewText = ""
		}

		edits = append(edits, edit)
	}

	return edits
}
