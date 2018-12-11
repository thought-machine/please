package langserver

import (
	"context"
	"core"
	"encoding/json"

	"github.com/sourcegraph/jsonrpc2"

	"github.com/thought-machine/please/src/parse/asp"
	"github.com/thought-machine/please/tools/build_langserver/lsp"
)

const renameMethod = "textDocument/rename"

func (h *LsHandler) handleRename(ctx context.Context, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, nil
	}

	var params lsp.RenameParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	documentURI, err := getURIAndHandleErrors(params.TextDocument.URI, renameMethod)
	if err != nil {
		return nil, err
	}

	edits, err := h.getRenameEdits(ctx, params.NewName, documentURI, params.Position)
	if err != nil {
		log.Warning("error occurred trying to get the rename edits from %s", documentURI)
	}

	return edits, nil
}

func (h *LsHandler) getRenameEdits(ctx context.Context, newName string,
	uri lsp.DocumentURI, pos lsp.Position) (*lsp.WorkspaceEdit, error) {

	renamingLabel, err := h.getRenamingLabel(ctx, uri, pos)
	if err != nil {
		return nil, err
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	doc, ok := h.workspace.documents[uri]
	if !ok {
		log.Info("document %s is not opened", uri)
		return nil, nil
	}
	workSpaceEdit := &lsp.WorkspaceEdit{
		Changes: make(map[lsp.DocumentURI][]lsp.TextEdit),
	}
	// Fill in workSpaceEdit.Changes first
	for _, label := range renamingLabel.RevDeps {
		buildLabel, err := h.analyzer.BuildLabelFromCoreBuildLabel(ctx, label)
		if err != nil {
			// In the case of error, we still return the current available locs
			return workSpaceEdit, nil
		}
		labelURI := lsp.DocumentURI("file://" + buildLabel.Path)
		edits := getEditsFromLabel(buildLabel, renamingLabel, newName)

		_, ok := workSpaceEdit.Changes[labelURI]
		if ok {
			workSpaceEdit.Changes[uri] = append(workSpaceEdit.Changes[uri], edits...)
		} else {
			workSpaceEdit.Changes[uri] = edits
		}
	}

	for k, v := range workSpaceEdit.Changes {
		docChange := lsp.TextDocumentEdit{
			TextDocument: lsp.VersionedTextDocumentIdentifier{
				TextDocumentIdentifier: &lsp.TextDocumentIdentifier{URI: k},
				Version:                doc.version,
			},
			Edits: v,
		}
		workSpaceEdit.DocumentChanges = append(workSpaceEdit.DocumentChanges, docChange)
	}

	return workSpaceEdit, nil
}

// getRenamingLabel returns a BuildLabel object with RevDeps being defined
func (h *LsHandler) getRenamingLabel(ctx context.Context, uri lsp.DocumentURI, pos lsp.Position) (*BuildLabel, error) {

	h.mu.Lock()
	defer h.mu.Unlock()

	def, err := h.analyzer.BuildDefsFromPos(ctx, uri, pos)
	if def == nil || err != nil || !isPosAtNameArg(def, pos) {
		return nil, err
	}

	coreLabel, err := getCoreBuildLabel(def, uri)
	if err != nil {
		return nil, err
	}
	renamingLabel, err := h.analyzer.BuildLabelFromCoreBuildLabel(ctx, coreLabel)
	if err != nil {
		return nil, err
	}

	revDeps, err := h.analyzer.RevDepsFromCoreBuildLabel(coreLabel, uri)
	if err != nil {
		log.Info("error occurred computing the reverse dependency of %s: %s", coreLabel.String(), err)
		return nil, err
	}

	renamingLabel.RevDeps = revDeps

	return renamingLabel, nil
}

func getEditsFromLabel(depLabel *BuildLabel, renaminglabel *BuildLabel, newName string) (edits []lsp.TextEdit) {
	if !isBuildDefValid(depLabel.BuildDef) {
		return nil
	}

	// Get the label string based on whether it's relative
	newLabelStr := core.NewBuildLabel(renaminglabel.PackageName, newName).String()
	oldLabelStr := renaminglabel.String()
	if renaminglabel.Path == depLabel.Path {
		newLabelStr = ":" + newName
		oldLabelStr = ":" + renaminglabel.Name
	}

	// Walk through the all the arguments from the call, and find string values that is the same as the old label string
	callback := func(astStruct interface{}) interface{} {
		if expr, ok := astStruct.(asp.Expression); ok && expr.Val != nil {
			if expr.Val.String != "" && TrimQuotes(expr.Val.String) == oldLabelStr {
				pos := aspPositionToLsp(expr.Pos)
				// Add 1 to the character, as we would be changing from inside of the quotes
				pos.Character = pos.Character + 1
				edit := lsp.TextEdit{
					Range:   getNameRange(pos, oldLabelStr),
					NewText: newLabelStr,
				}
				edits = append(edits, edit)
			}
		}

		return nil
	}

	asp.WalkAST(depLabel.BuildDef.Action.Call.Arguments, callback)

	return edits
}

func isPosAtNameArg(def *BuildDef, pos lsp.Position) bool {

	if !isBuildDefValid(def) {
		return false
	}

	args := def.Action.Call.Arguments

	for _, arg := range args {
		if arg.Name == "name" && withInRange(arg.Value.Pos, arg.Value.EndPos, pos) {
			return true
		}
	}

	return false
}

func isBuildDefValid(def *BuildDef) bool {
	if def.Action == nil || def.Action.Call == nil ||
		def.Action.Call.Arguments == nil || len(def.Action.Call.Arguments) == 0 {

		return false
	}

	return true
}
