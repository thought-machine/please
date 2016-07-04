package main

import (
	"gopkg.in/op/go-logging.v1"

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
	LogFile        string          `long:"log_file" description:"File to log to (in addition to stdout)"`
	KeyFile        string          `long:"key_file" description:"File containing PEM-encoded private key."`
	CertFile       string          `long:"cert_file" description:"File containing PEM-encoded certificate"`
	CACertFile     string          `long:"ca_cert_file" description:"File containing PEM-encoded CA certificate"`
	WritableCerts  string          `long:"writable_certs" description:"File or directory containing certificates that are allowed to write to the cache"`
	ReadonlyCerts  string          `long:"readonly_certs" description:"File or directory containing certificates that are allowed to read from the cache"`
}

func main() {
	output.ParseFlagsOrDie("Please RPC cache server", &opts)
	output.InitLogging(opts.Verbosity, opts.LogFile, opts.Verbosity)
	if (opts.KeyFile == "") != (opts.CertFile == "") {
		log.Fatalf("Must pass both --key_file and --cert_file if you pass one")
	} else if opts.KeyFile == "" && (opts.WritableCerts != "" || opts.ReadonlyCerts != "") {
		log.Fatalf("You can only use --writable_certs / --readonly_certs with https (--key_file and --cert_file)")
	}
	log.Notice("Scanning existing cache directory %s...", opts.Dir)
	cache := server.NewCache(opts.Dir, opts.CleanFrequency, int64(opts.LowWaterMark), int64(opts.HighWaterMark))
	log.Notice("Starting up RPC cache server on port %d...", opts.Port)
	server.ServeGrpcForever(opts.Port, cache, opts.KeyFile, opts.CertFile, opts.CACertFile, opts.ReadonlyCerts, opts.WritableCerts)
}
