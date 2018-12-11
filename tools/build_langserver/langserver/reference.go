package langserver

import (
	"context"
	"encoding/json"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/thought-machine/please/tools/build_langserver/lsp"
)

const referenceMethod = "textDocument/references"

func (h *LsHandler) handleReferences(ctx context.Context, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, nil
	}

	var params lsp.ReferenceParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	documentURI, err := getURIAndHandleErrors(params.TextDocument.URI, referenceMethod)
	if err != nil {
		return nil, err
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	refs, err := h.getReferences(ctx, documentURI, params.Position)
	if err != nil && len(refs) == 0 {
		log.Warning("error occurred when trying to get references: %s", err)
		return nil, nil
	}

	return refs, nil
}

func (h *LsHandler) getReferences(ctx context.Context, uri lsp.DocumentURI, pos lsp.Position) ([]lsp.Location, error) {
	def, err := h.analyzer.BuildDefsFromPos(ctx, uri, pos)
	if def == nil || err != nil || def.Pos.Line != pos.Line {
		return nil, err
	}

	revDeps, err := h.analyzer.RevDepsFromBuildDef(def, uri)
	if err != nil {
		log.Info("error occurred computing the reverse dependency of %")
		return nil, err
	}

	var locs []lsp.Location
	for _, label := range revDeps {
		buildLabel, err := h.analyzer.BuildLabelFromCoreBuildLabel(ctx, label)
		if err != nil {
			// In the case of error, we still return the current available locs
			return locs, nil
		}

		loc := lsp.Location{
			URI: lsp.DocumentURI("file://" + buildLabel.Path),
			Range: lsp.Range{
				Start: buildLabel.BuildDef.Pos,
				End:   buildLabel.BuildDef.EndPos,
			},
		}
		locs = append(locs, loc)
	}

	return locs, nil
}
