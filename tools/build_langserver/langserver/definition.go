package langserver

import (
	"context"
	"github.com/thought-machine/please/tools/build_langserver/lsp"

	"github.com/sourcegraph/jsonrpc2"
)

const definitionMethod = "textDocument/definition"

func (h *LsHandler) handleDefinition(ctx context.Context, req *jsonrpc2.Request) (result interface{}, err error) {
	params, err := h.getParamFromTDPositionReq(req, definitionMethod)
	if err != nil {
		return nil, err
	}
	documentURI := params.TextDocument.URI

	h.mu.Lock()
	defer h.mu.Unlock()
	definitionLocations := h.getDefinitionLocation(ctx, documentURI, params.Position)
	log.Info("definition locations: %s", definitionLocations)

	return definitionLocations, nil
}

func (h *LsHandler) getDefinitionLocation(ctx context.Context, uri lsp.DocumentURI, pos lsp.Position) []lsp.Location {
	fileContent := h.workspace.documents[uri].textInEdit
	lineContent := h.ensureLineContent(uri, pos)

	log.Info("goto definition lineContent: %s", lineContent)

	if isEmpty(lineContent, pos) {
		return nil
	}

	stmts := h.analyzer.AspStatementFromContent(JoinLines(fileContent, true))

	buildLabel := h.analyzer.BuildLabelFromAST(ctx, stmts, uri, pos)

	if buildLabel != nil {
		uri, err := EnsureURL(lsp.DocumentURI(buildLabel.Path), "file")
		if err != nil {
			log.Error("fail to get file: %s", buildLabel.Path)
			return nil
		}

		if buildLabel.BuildDef != nil {
			loc := lsp.Location{
				URI: uri,
				Range: lsp.Range{
					Start: buildLabel.BuildDef.Pos,
					End:   buildLabel.BuildDef.EndPos,
				},
			}
			return []lsp.Location{loc}
		}
	}

	return nil
}
