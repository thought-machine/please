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
	ctx := context.Background()
	filepath := path.Join(core.RepoRoot, "tools/build_langserver/langserver/BUILD")
	uri := lsp.DocumentURI("file://" + filepath)
	position := lsp.Position{
		Line: 18,
		Character: 3,
	}



}