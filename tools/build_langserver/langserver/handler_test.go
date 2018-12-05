package langserver

import (
	"context"
	"github.com/thought-machine/please/src/core"
	"github.com/sourcegraph/jsonrpc2"
	"github.com/stretchr/testify/assert"
	"net"
	"strings"
	"testing"
	"github.com/thought-machine/please/tools/build_langserver/lsp"
)

// TODO(bnm): cleanup

var bindAddr = "127.0.0.1:4040"

func TestHandle(t *testing.T) {
	h := NewHandler()
	lis := startServer(t, h)
	defer lis.Close()
	conn := dialServer(t, bindAddr)
	defer func() {
		if err := conn.Close(); err != nil {
			t.Fatal("conn.Close:", err)
		}
	}()

	ctx := context.Background()
	tdCap := lsp.TextDocumentClientCapabilities{}

	var result lsp.InitializeResult
	if err := conn.Call(ctx, "initialize", lsp.InitializeParams{
		RootURI:      lsp.DocumentURI(core.RepoRoot),
		Capabilities: lsp.ClientCapabilities{TextDocument: tdCap},
	}, &result); err != nil {
		t.Fatal("initialize:", err)
	}
	t.Log(result)

	var result2 lsp.Hover
	uri := lsp.DocumentURI(exampleBuildURI)
	openFile(ctx, t, conn, uri)

	if err := conn.Call(ctx, "textDocument/hover", lsp.TextDocumentPositionParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: lsp.DocumentURI(exampleBuildURI),
		},
		Position: lsp.Position{Line: 0, Character: 3},
	}, &result2); err != nil {
		t.Fatal("hover:", err)
	}
	t.Log(result2)
}

/*
 * Utilities function for tests in this file
 */
func startServer(t testing.TB, h jsonrpc2.Handler) net.Listener {
	listener, err := net.Listen("tcp", bindAddr)
	if err != nil {
		t.Fatal("Listen:", err)
	}
	go func() {
		if err := serve(context.Background(), listener, h); err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
			t.Fatal("jsonrpc2.Serve:", err)
		}
	}()
	return listener
}

func serve(ctx context.Context, lis net.Listener, h jsonrpc2.Handler) error {
	for {
		conn, err := lis.Accept()
		if err != nil {
			return err
		}
		jsonrpc2.NewConn(ctx, jsonrpc2.NewBufferedStream(conn, jsonrpc2.VSCodeObjectCodec{}), h)
	}
}

func dialServer(t testing.TB, addr string) *jsonrpc2.Conn {
	conn, err := (&net.Dialer{}).Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}

	handler := jsonrpc2.HandlerWithError(func(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (interface{}, error) {
		return nil, nil
	})

	return jsonrpc2.NewConn(
		context.Background(),
		jsonrpc2.NewBufferedStream(conn, jsonrpc2.VSCodeObjectCodec{}),
		handler,
	)
}

func openFile(ctx context.Context, t testing.TB, conn *jsonrpc2.Conn, uri lsp.DocumentURI) {
	content, err := ReadFile(ctx, uri)
	assert.Equal(t, nil, err)
	text := strings.Join(content, "\n")

	if err := conn.Call(ctx, "textDocument/didOpen", lsp.DidOpenTextDocumentParams{
		TextDocument: lsp.TextDocumentItem{
			URI:        uri,
			LanguageID: "build",
			Version:    1,
			Text:       text,
		},
	}, nil); err != nil {
		t.Fatal("open failure:", err)
	}
}
