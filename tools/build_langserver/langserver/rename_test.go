package langserver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/tools/build_langserver/lsp"
)

func TestGetRenameEdits(t *testing.T) {
	ctx := context.Background()

	// copy over the handler from the setup and get a new analyzer so it would be reading the config
	a, _ := newAnalyzer()
	h := LsHandler{
		repoRoot:  core.RepoRoot,
		workspace: newWorkspaceStore(lsp.DocumentURI(core.RepoRoot)),
		analyzer:  a,
		init: &lsp.InitializeParams{
			RootURI: lsp.DocumentURI(core.RepoRoot),
			Capabilities: lsp.ClientCapabilities{
				TextDocument: lsp.TextDocumentClientCapabilities{
					Completion: lsp.Completion{
						CompletionItem: struct {
							SnippetSupport bool `json:"snippetSupport,omitempty"`
						}{SnippetSupport: true}},
				},
			},
		},
	}

	uri := lsp.DocumentURI("file://tools/build_langserver/langserver/test_data/reference/BUILD.test")
	h.analyzer.State.Config.Parse.BuildFileName = append(analyzer.State.Config.Parse.BuildFileName,
		"BUILD.test")
	storeFileWithCustomHandler(uri, &h)

	edits, err := h.getRenameEdits(ctx, "blah", uri, lsp.Position{Line: 1, Character: 13})
	assert.NoError(t, err)

	expected := lsp.Range{
		Start: lsp.Position{Line: 12, Character: 9},
		End:   lsp.Position{Line: 12, Character: 15},
	}

	// Check WorkspaceEdit.Changes
	assert.Equal(t, 1, len(edits.Changes))
	eRange, ok := edits.Changes[uri]
	assert.True(t, ok)
	assert.Equal(t, 1, len(eRange))
	assert.Equal(t, expected, eRange[0].Range)
	assert.Equal(t, ":blah", eRange[0].NewText)

	// Check WorkspaceEdit.DocumentChanges
	assert.Equal(t, 1, len(edits.DocumentChanges))
	assert.Equal(t, uri, edits.DocumentChanges[0].TextDocument.URI)
	assert.Equal(t, 1, edits.DocumentChanges[0].TextDocument.Version)
	assert.Equal(t, expected, edits.DocumentChanges[0].Edits[0].Range)
	assert.Equal(t, ":blah", edits.DocumentChanges[0].Edits[0].NewText)
}
