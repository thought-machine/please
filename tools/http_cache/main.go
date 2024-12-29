package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/thought-machine/please/src/cli"
	logger "github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/tools/http_cache/cache"
)

var log = logger.Log

var opts = struct {
	Usage     string
	Verbosity cli.Verbosity `short:"v" long:"verbosity" default:"notice" description:"Verbosity of output (higher number = more output)"`
	CacheDir  string        `short:"d" long:"dir" default:"" description:"The directory to store cached artifacts in."`
	Port      int           `short:"p" long:"port" description:"The port to run the server on" default:"8080"`
}{
	Usage: `
HTTP cache implements a resource based http server that please can use as a cache. The cache supports storing files
via PUT requests and retrieving them again through GET requests. Really any http server (e.g. nginx) can be used as a
cache for please however this is a lightweight and easy to configure option.
`,
}

func main() {
	cli.ParseFlagsOrDie("HTTP Cache", &opts)
	cli.InitLogging(opts.Verbosity)

	if opts.CacheDir == "" {
		userCacheDir, err := os.UserCacheDir()
		if err != nil {
			log.Fatalf("failed to get user cache dir: %v", err)
		}
		opts.CacheDir = filepath.Join(userCacheDir, "please_http_cache")
	}

	log.Notice("Started please http cache at 127.0.0.1:%v serving out of %v", opts.Port, opts.CacheDir)
	err := http.ListenAndServe(fmt.Sprint(":", opts.Port), cache.New(opts.CacheDir))
	if err != nil {
		log.Panic(err)
	}
}
