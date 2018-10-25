package langserver

import (
	"context"
	"path"
	"testing"

	"tools/build_langserver/lsp"

	"github.com/stretchr/testify/assert"
)

var completionPath = path.Join("tools/build_langserver/langserver/test_data/completion.build")
var completionURI = lsp.DocumentURI("file://" + completionPath)

func TestCompletion(t *testing.T) {
	ctx := context.Background()

	items, err := getCompletionItems(ctx, analyzer, true,
		completionURI, lsp.Position{Line: 0, Character: 6})

	assert.Equal(t, nil, err)
	t.Log(items)
}
