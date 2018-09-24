package main

import (
	"context"
	"gopkg.in/op/go-logging.v1"
	"os"

	"github.com/sourcegraph/jsonrpc2"
	"tools/build_langserver/langserver"
)

var log = logging.MustGetLogger("build_langserver")

func main() {
	handler := langserver.NewHandler

	// TODO(luna): make this work!!
	log.Info("build_langserver: reading on stdin, writing on stdout")
	//jsonrpc2.NewConn(context.Background(), handler, interface{})
	log.Info("connection closed")
	return nil
}

type stdrwc struct{}

func (stdrwc) Read(p []byte) (int, error) {
	return os.Stdin.Read(p)
}

func (stdrwc) Write(p []byte) (int, error) {
	return os.Stdout.Write(p)
}

func (stdrwc) Close() error {
	if err := os.Stdin.Close(); err != nil {
		return err
	}
	return os.Stdout.Close()
}
