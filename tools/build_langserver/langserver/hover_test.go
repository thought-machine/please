package langserver

//import (
//	"context"
//	"core"
//	"path"
//	"testing"
//	"tools/build_langserver/lsp"
//
//)
//
//func TestGetHoverContent(t *testing.T) {
//	core.FindRepoRoot()
//	ctx := context.Background()
//	filepath := path.Join(core.RepoRoot, "tools/build_langserver/langserver/BUILD")
//	uri := lsp.DocumentURI("file://" + filepath)
//	analyzer := newAnalyzer()
//	position := lsp.Position{
//		Line: 18,
//		Character: 3,
//	}
//
//	_, err := getHoverContent(ctx, analyzer, uri, position)
//	t.Log(err)
//	t.Log(filepath)
//}