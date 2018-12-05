package langserver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/thought-machine/please/tools/build_langserver/lsp"

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

	log.Info("formatting document with edits: %s", edits)
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
	beforeLines := SplitLines(before, true)
	afterLines := SplitLines(after, true)

	var edits []*lsp.TextEdit
	for i, line := range afterLines {

		eRange := lsp.Range{
			Start: lsp.Position{
				Line:      i,
				Character: 0,
			},
			End: lsp.Position{
				Line:      i,
				Character: 0,
			},
		}

		if i <= len(beforeLines)-1 {
			if line == beforeLines[i] {
				continue
			}

			if i < len(beforeLines)-1 {
				eRange.End.Line = i + 1
			} else if beforeLines[i] != "" {
				// This is to ensure the original line gets overridden if there is no newline after:
				// usually if the last line is new line, it gets formatted correctly with startline:currentline, endline:currline + 1
				// however this does not apply we want to format the last line
				eRange.End.Character = len(beforeLines[i]) - 1
			}
		}

		edit := &lsp.TextEdit{
			Range:   eRange,
			NewText: line,
		}

		edits = append(edits, edit)
	}

	return edits
}
