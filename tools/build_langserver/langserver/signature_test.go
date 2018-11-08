package langserver

import (
	"context"
	"testing"

	"tools/build_langserver/lsp"

	"github.com/stretchr/testify/assert"
)

func TestGetSignatures(t *testing.T) {
	ctx := context.Background()

	err := storeFile(ctx, sigURI)
	assert.Equal(t, nil, err)

	sig, err := handler.getSignatures(ctx, sigURI, lsp.Position{Line: 0, Character: 10})
	assert.Equal(t, nil, err)
	t.Log(sig)
}
