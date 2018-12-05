package langserver

import (
	"context"
	"github.com/thought-machine/please/src/core"
	"testing"

	"github.com/thought-machine/please/tools/build_langserver/lsp"

	"github.com/stretchr/testify/assert"
	"os"
	"path"
)

func TestGetReferences(t *testing.T) {
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

	h.analyzer.State.Config.Parse.BuildFileName = append(analyzer.State.Config.Parse.BuildFileName,
		"BUILD.test")

	testDir, err := os.Getwd()
	assert.NoError(t, err)
	uri := lsp.DocumentURI("file://tools/build_langserver/langserver/test_data/reference/BUILD.test")

	locs, err := h.getReferences(ctx, uri, lsp.Position{Line: 0, Character: 4})
	assert.NoError(t, err)

	assert.Equal(t, 1, len(locs))

	// Reference in //src:please
	expected := lsp.Location{
		URI: lsp.DocumentURI("file://" + path.Join(testDir, "tools/build_langserver/langserver/test_data/reference/BUILD.test")),
		Range: lsp.Range{
			Start: lsp.Position{Line: 8, Character: 0},
			End:   lsp.Position{Line: 14, Character: 1},
		},
	}
	assertLocInList(t, locs, expected)
}

func assertLocInList(t testing.TB, locs []lsp.Location, passLoc lsp.Location) {
	for _, loc := range locs {
		if loc.URI == passLoc.URI {
			assert.Equal(t, loc.Range, passLoc.Range)
			return
		}
	}
	// mark test as fail if not found
	t.Errorf("loc %s is not in the list", passLoc)
}
