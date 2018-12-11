package main

import (
	"fmt"
	"net/http"
	"time"

	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/tools/cache/server"
)

var log = logging.MustGetLogger("http_cache_server")

var opts struct {
	Usage     string        `usage:"http_cache_server is a server for Please's remote HTTP cache.\n\nSee https://please.build/cache.html for more information."`
	Verbosity cli.Verbosity `short:"v" long:"verbosity" default:"notice" description:"Verbosity of output (higher number = more output)"`
	Port      int           `short:"p" long:"port" description:"Port to serve on" default:"8080"`
	Dir       string        `short:"d" long:"dir" description:"Directory to write into" default:"plz-http-cache"`
	LogFile   string        `long:"log_file" description:"File to log to (in addition to stdout)"`

	CleanFlags struct {
		LowWaterMark   cli.ByteSize `short:"l" long:"low_water_mark" description:"Size of cache to clean down to" default:"18G"`
		HighWaterMark  cli.ByteSize `short:"i" long:"high_water_mark" description:"Max size of cache to clean at" default:"20G"`
		CleanFrequency cli.Duration `short:"f" long:"clean_frequency" description:"Frequency to clean cache at" default:"10m"`
		MaxArtifactAge cli.Duration `short:"m" long:"max_artifact_age" description:"Clean any artifact that's not been read in this long" default:"720h"`
	} `group:"Options controlling when to clean the cache"`
}

func main() {
	cli.ParseFlagsOrDie("Please HTTP cache server", "5.5.0", &opts)
	cli.InitLogging(opts.Verbosity)
	if opts.LogFile != "" {
		cli.InitFileLogging(opts.LogFile, opts.Verbosity)
	}
	log.Notice("Initialising cache server...")
	cache := server.NewCache(opts.Dir, time.Duration(opts.CleanFlags.CleanFrequency),
		time.Duration(opts.CleanFlags.MaxArtifactAge),
		uint64(opts.CleanFlags.LowWaterMark), uint64(opts.CleanFlags.HighWaterMark))
	log.Notice("Starting up http cache server on port %d...", opts.Port)
	router := server.BuildRouter(cache)
	http.Handle("/", router)
	http.ListenAndServe(fmt.Sprintf(":%d", opts.Port), router)
}
