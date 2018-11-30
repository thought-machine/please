package langserver

import (
	"context"
	"core"
	"testing"

	"tools/build_langserver/lsp"
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
	storeFile(uri)

	h.getRenameEdits(ctx, "blah", uri, lsp.Position{Line: 1, Character: 13})
}
