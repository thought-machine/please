package main

import (
	"github.com/op/go-logging"

	"cache/server"
	"output"
)

var log = logging.MustGetLogger("rpc_cache_server")

var opts struct {
	Verbosity      int             `short:"v" long:"verbosity" description:"Verbosity of output (higher number = more output, default 2 -> notice, warnings and errors only)" default:"2"`
	Port           int             `short:"p" long:"port" description:"Port to serve on" default:"7677"`
	Dir            string          `short:"d" long:"dir" description:"Directory to write into" default:"plz-rpc-cache"`
	LowWaterMark   output.ByteSize `short:"l" long:"low_water_mark" description:"Size of cache to clean down to" default:"18G"`
	HighWaterMark  output.ByteSize `short:"i" long:"high_water_mark" description:"Max size of cache to clean at" default:"20G"`
	CleanFrequency int             `short:"f" long:"clean_frequency" description:"Frequency to clean cache at, in seconds" default:"10"`
}

func main() {
	output.ParseFlagsOrDie("Please RPC cache server", &opts)
	output.InitLogging(opts.Verbosity, "", 0)
	log.Notice("Scanning existing cache directory %s...", opts.Dir)
	server.Init(opts.Dir, opts.CleanFrequency, int64(opts.LowWaterMark), int64(opts.HighWaterMark))
	log.Notice("Starting up RPC cache server on port %d...", opts.Port)
	server.ServeGrpcForever(opts.Port)
}
