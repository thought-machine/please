package langserver

import (
	"context"
	"core"
	"path"
	"testing"
	"tools/build_langserver/lsp"

	"github.com/stretchr/testify/assert"
)

func TestGetHoverContent(t *testing.T) {
	core.FindRepoRoot()
	ctx := context.Background()
	filepath := path.Join(core.RepoRoot, "tools/build_langserver/langserver/test_data/example.build")
	uri := lsp.DocumentURI("file://" + filepath)

	analyzer := newAnalyzer()
	// Test hovering on the function call
	content, err := getHoverContent(ctx, analyzer, uri, lsp.Position{Line: 0, Character: 3})
	expected := analyzer.BuiltIns["go_library"].Header + "\n\n"  + analyzer.BuiltIns["go_library"].Docstring

	assert.Equal(t, nil, err)
	assert.Equal(t, expected, content.Value)

	// Test hovering over arguments
	content, err = getHoverContent(ctx, analyzer, uri, lsp.Position{Line:7, Character:7})
	assert.Equal(t, nil, err)
	assert.Equal(t, "deps required:false, type:list", content.Value)

	// Test hovering over nexted call
	content, err = getHoverContent(ctx, analyzer, uri, lsp.Position{Line:5, Character:10})
	assert.Equal(t, nil, err)
	assert.Equal(t, "def glob(include:list, exclude:list&excludes=[], hidden:bool=False)\n\n",
		content.Value)

	// Test hovering over argument of nexted call
	content, err = getHoverContent(ctx, analyzer, uri, lsp.Position{Line:4, Character:15})
	assert.Equal(t, nil, err)
	assert.Equal(t, "exclude required:false, type:list",
		content.Value)
}
