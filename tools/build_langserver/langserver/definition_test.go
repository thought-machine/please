package langserver

import (
	"context"
	"github.com/stretchr/testify/assert"
	"path"
	"src/core"
	"testing"

	"github.com/thought-machine/please/tools/build_langserver/lsp"
)

func TestGetDefinitionLocationOnBuildDefs(t *testing.T) {
	ctx := context.Background()
	core.FindRepoRoot()

	// GOTO definition on buildDef deps (complete build label)
	loc := handler.getDefinitionLocation(ctx, exampleBuildURI, lsp.Position{Line: 9, Character: 14})

	expectedURI := lsp.DocumentURI("file://" + path.Join(core.RepoRoot, "src/fs/BUILD"))
	expectedRange := lsp.Range{
		Start: lsp.Position{Line: 0, Character: 0},
		End:   lsp.Position{Line: 15, Character: 1},
	}
	assert.Equal(t, 1, len(loc))
	assert.Equal(t, expectedURI, loc[0].URI)
	assert.Equal(t, expectedRange, loc[0].Range)

	// GOTO definition on third_party dir
	loc = handler.getDefinitionLocation(ctx, exampleBuildURI, lsp.Position{Line: 13, Character: 21})
	expectedURI = lsp.DocumentURI("file://" + path.Join(core.RepoRoot, "third_party/go/BUILD"))
	expectedRange = lsp.Range{
		Start: lsp.Position{Line: 617, Character: 0},
		End:   lsp.Position{Line: 621, Character: 1},
	}
	assert.Equal(t, 1, len(loc))
	assert.Equal(t, expectedURI, loc[0].URI)
	assert.Equal(t, expectedRange, loc[0].Range)

	// GOTO definition buildlabel for including all subpackages
	// *Should return nil*
	loc = handler.getDefinitionLocation(ctx, exampleBuildURI, lsp.Position{Line: 6, Character: 29})
	assert.Equal(t, 0, len(loc))

	// GOTO definition on list on the same line
	loc = handler.getDefinitionLocation(ctx, exampleBuildURI, lsp.Position{Line: 6, Character: 58})
	expectedURI = lsp.DocumentURI("file://" + path.Join(core.RepoRoot, "src/core/BUILD"))
	expectedRange = lsp.Range{
		Start: lsp.Position{Line: 10, Character: 0},
		End:   lsp.Position{Line: 34, Character: 1},
	}
	assert.Equal(t, 1, len(loc))
	assert.Equal(t, expectedURI, loc[0].URI)
	assert.Equal(t, expectedRange, loc[0].Range)

	// GOTO definition on relative label
	analyzer.State.Config.Parse.BuildFileName = []string{"example.build", "BUILD"}
	loc = handler.getDefinitionLocation(ctx, exampleBuildURI, lsp.Position{Line: 27, Character: 16})
	expectedRange = lsp.Range{
		Start: lsp.Position{Line: 0, Character: 0},
		End:   lsp.Position{Line: 17, Character: 1},
	}
	assert.Equal(t, 1, len(loc))
	//assert.Equal(t, exampleBuildURI, loc[0].URI)
	assert.Equal(t, expectedRange, loc[0].Range)
}

func TestGetDefinitionLocationOnAssignments(t *testing.T) {
	ctx := context.Background()

	// GOTO definition on assignment list
	loc := handler.getDefinitionLocation(ctx, assignBuildURI, lsp.Position{Line: 19, Character: 26})
	expectedURI := lsp.DocumentURI("file://" + path.Join(core.RepoRoot, "src/fs/BUILD"))
	expectedRange := lsp.Range{
		Start: lsp.Position{Line: 0, Character: 0},
		End:   lsp.Position{Line: 15, Character: 1},
	}
	assert.Equal(t, 1, len(loc))
	assert.Equal(t, expectedURI, loc[0].URI)
	assert.Equal(t, expectedRange, loc[0].Range)

	loc = handler.getDefinitionLocation(ctx, assignBuildURI, lsp.Position{Line: 19, Character: 41})
	expectedURI = lsp.DocumentURI("file://" + path.Join(core.RepoRoot, "src/core/BUILD"))
	expectedRange = lsp.Range{
		Start: lsp.Position{Line: 10, Character: 0},
		End:   lsp.Position{Line: 34, Character: 1},
	}
	assert.Equal(t, 1, len(loc))
	assert.Equal(t, expectedURI, loc[0].URI)
	assert.Equal(t, expectedRange, loc[0].Range)

	// GOTO definition on assignment string
	loc = handler.getDefinitionLocation(ctx, assignBuildURI, lsp.Position{Line: 25, Character: 17})
	expectedURI = lsp.DocumentURI("file://" + path.Join(core.RepoRoot, "src/fs/BUILD"))
	expectedRange = lsp.Range{
		Start: lsp.Position{Line: 0, Character: 0},
		End:   lsp.Position{Line: 15, Character: 1},
	}
	assert.Equal(t, 1, len(loc))
	assert.Equal(t, expectedURI, loc[0].URI)
	assert.Equal(t, expectedRange, loc[0].Range)
}

func TestGetDefinitionOnSubIncludeArgument(t *testing.T) {
	ctx := context.Background()

	loc := handler.getDefinitionLocation(ctx, assignBuildURI, lsp.Position{Line: 6, Character: 33})
	expectedURI := lsp.DocumentURI("file://" + path.Join(core.RepoRoot, "build_defs/BUILD"))
	expectedRange := lsp.Range{
		Start: lsp.Position{Line: 6, Character: 0},
		End:   lsp.Position{Line: 10, Character: 1},
	}
	assert.Equal(t, 1, len(loc))
	assert.Equal(t, expectedURI, loc[0].URI)
	assert.Equal(t, expectedRange, loc[0].Range)
}
