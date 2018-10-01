package main

import (
	"context"
	"gopkg.in/op/go-logging.v1"
	"os"

	"cli"
	"github.com/sourcegraph/jsonrpc2"
	"tools/build_langserver/langserver"
)

// TODO(bnmetrics): also think about how we can implement this with .build_defs as well
// TODO(bnmetrics): Make the Usage part better
var log = logging.MustGetLogger("build_langserver")

var opts = struct {
	Usage     string
	Verbosity cli.Verbosity `short:"v" long:"verbosity" default:"warning" description:"Verbosity of output (higher number = more output)"`

	Mode string `short:"m" long:"mode" default:"stdio" choice:"stdio" choice:"tcp" description:"Mode of the language server communication"`
	Host string `short:"h" long:"host" default:"127.0.0.20" description:"TCP host to communicate with"`
	Port string `short:"p" long:"port" default:"4387" description:"TCP port to communicate with"`
}{
	Usage: `
build_langserver is a binary shipped with Please that you can use as a language server for build files.

It speaks language server protocol from vscode, you can plugin this binary in your IDE to start the language server.
Currently, it supports autocompletion, goto definition for build_defs, and signature help
`,
}

func main() {
	cli.ParseFlagsOrDie("build_langserver", "1.0.0", &opts)
	cli.InitLogging(opts.Verbosity)

	handler := langserver.NewHandler()

	serve(handler)
}

func serve(handler jsonrpc2.Handler) {
	if opts.Mode == "tcp" {
		// TODO: tcp stuff
	} else {
		log.Info("build_langserver: reading on stdin, writing on stdout")

		<-jsonrpc2.NewConn(context.Background(), jsonrpc2.NewBufferedStream(stdrwc{}, jsonrpc2.VSCodeObjectCodec{}),
			handler, []jsonrpc2.ConnOpt{}...).DisconnectNotify()

		log.Info("connection closed")
	}
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
