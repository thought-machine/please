package langserver

import (
	"build"
	"context"
	"core"
	"encoding/json"
	"fmt"
	"parse"
	"path/filepath"
	"plz"
	"query"

	"tools/build_langserver/lsp"

	"github.com/sourcegraph/jsonrpc2"
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
	if def == nil || err != nil {
		return nil, err
	}

	label, err := getCoreBuildLabel(def, uri)
	if err != nil {
		return nil, err
	}

	//Ensure we do not get locked out
	h.analyzer.State.NeedBuild = false
	h.analyzer.State.NeedTests = false

	parse.InitParser(h.analyzer.State)
	build.Init(h.analyzer.State)
	success, state := plz.InitDefault([]core.BuildLabel{label}, h.analyzer.State,
		h.analyzer.State.Config)

	if !success {
		log.Warning("building %s not successful, skipping..", label)
		return nil, nil
	}
	revDeps := query.GetRevDepsLabels(state, []core.BuildLabel{label})

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

func getCoreBuildLabel(def *BuildDef, uri lsp.DocumentURI) (core.BuildLabel, error) {
	fp, err := GetPathFromURL(uri, "file")
	if err != nil {
		return core.BuildLabel{}, err
	}

	rel, err := filepath.Rel(core.RepoRoot, filepath.Dir(fp))
	if err != nil {
		return core.BuildLabel{}, err
	}

	labelStr := fmt.Sprintf("//%s:%s", rel, def.BuildDefName)

	return core.TryParseBuildLabel(labelStr, fp)
}
