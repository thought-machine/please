package langserver

import (
	"os"

	"context"
	"core"
	"path"
	"testing"

	"tools/build_langserver/lsp"

	"github.com/stretchr/testify/assert"
)

func TestGetReferences(t *testing.T) {
	ctx := context.Background()

	// copy over the handler from the setup and get a new analyzer so it would be reading the config
	a, _ := newAnalyzer()
	h := handler
	h.analyzer = a

	uri := lsp.DocumentURI(path.Join(core.RepoRoot, "src/query/BUILD"))

	// TODO(bnm): We need to think of a better way to run handler.getReferences in root directory instead of Chdir here
	// save the current testing directory
	testDir, err := os.Getwd()
	assert.NoError(t, err)

	// Change to repoRoot before we run the test
	err = os.Chdir(core.RepoRoot)
	assert.NoError(t, err)

	locs, err := h.getReferences(ctx, uri, lsp.Position{Line: 0, Character: 4})
	assert.NoError(t, err)

	err = os.Chdir(testDir)
	assert.NoError(t, err)

	assert.Equal(t, 6, len(locs))

	// Reference in //src:please
	expected := lsp.Location{
		URI: lsp.DocumentURI("file://" + path.Join(core.RepoRoot, "src/BUILD.plz")),
		Range: lsp.Range{
			Start: lsp.Position{Line: 0, Character: 0},
			End:   lsp.Position{Line: 32, Character: 1},
		},
	}
	assertLocInList(t, locs, expected)

	// reference in query
	queryURI := lsp.DocumentURI("file://" + path.Join(core.RepoRoot, "src/query/BUILD"))
	var refCount int
	for _, loc := range locs {
		if loc.URI == queryURI {
			refCount++
		}
	}
	assert.Equal(t, 4, refCount)

	// reference in langserver
	expected = lsp.Location{
		URI: lsp.DocumentURI("file://" + path.Join(core.RepoRoot, "tools/build_langserver/langserver/BUILD")),
		Range: lsp.Range{
			Start: lsp.Position{Line: 0, Character: 0},
			End:   lsp.Position{Line: 22, Character: 1},
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
