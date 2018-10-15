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
	position := lsp.Position{
		Line: 7,
		Character: 7,
	}

	content, err := getHoverContent(ctx, analyzer, uri, position)
	assert.Equal(t, nil, err)
	assert.Equal(t, "deps required:false, type:list", content.Value)
}