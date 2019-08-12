package main

import (
	"context"
	"io"
	"net"
	"os"

	"github.com/sourcegraph/jsonrpc2"
	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/tools/build_langserver/lsp"
)

var log = logging.MustGetLogger("build_langserver")

var opts = struct {
	Usage     string
	Verbosity cli.Verbosity `short:"v" long:"verbosity" default:"notice" description:"Verbosity of output (higher number = more output)"`
	LogFile   cli.Filepath  `long:"log_file" description:"File to echo full logging output to"`

	Mode string `short:"m" long:"mode" default:"stdio" choice:"stdio" choice:"tcp" description:"Mode of the language server communication"`
	Host string `short:"h" long:"host" default:"127.0.0.1" description:"TCP host to communicate with"`
	Port string `short:"p" long:"port" default:"4040" description:"TCP port to communicate with"`
}{
	Usage: `
build_langserver is a binary shipped with Please that you can use as a language server for build files.

It speaks language server protocol from vscode, you can plugin this binary in your IDE to start the language server.
Currently, it supports autocompletion, goto definition for build_defs, and signature help
`,
}

func main() {
	cli.ParseFlagsOrDie("build_langserver", &opts)
	cli.InitLogging(opts.Verbosity)
	if opts.LogFile != "" {
		cli.InitFileLogging(string(opts.LogFile), opts.Verbosity)
	}
	if err := serve(lsp.NewHandler()); err != nil {
		log.Fatalf("fail to start server: %s", err)
	}
}

func serve(handler *lsp.Handler) error {
	if opts.Mode == "tcp" {
		lis, err := net.Listen("tcp", opts.Host+":"+opts.Port)
		if err != nil {
			return err
		}
		defer lis.Close()
		log.Notice("build_langserver: listening on %s:%s", opts.Host, opts.Port)
		conn, err := lis.Accept()
		if err != nil {
			return err
		}
		reallyServe(handler, conn)
	} else {
		log.Info("build_langserver: reading on stdin, writing on stdout")
		reallyServe(handler, stdrwc{})
	}
	return nil
}

func reallyServe(handler *lsp.Handler, conn io.ReadWriteCloser) {
	c := jsonrpc2.NewConn(
		context.Background(),
		jsonrpc2.NewBufferedStream(conn, jsonrpc2.VSCodeObjectCodec{}),
		handler,
		jsonrpc2.LogMessages(lsp.Logger{}),
	)
	handler.Conn = c
	<-c.DisconnectNotify()
	log.Info("connection closed")
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
