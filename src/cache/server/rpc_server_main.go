package main

import (
	"strings"
	"time"

	"gopkg.in/op/go-logging.v1"

	"cache/server"
	"cli"
)

var log = logging.MustGetLogger("rpc_cache_server")

var opts struct {
	Port      int    `short:"p" long:"port" description:"Port to serve on" default:"7677"`
	Dir       string `short:"d" long:"dir" description:"Directory to write into" default:"plz-rpc-cache"`
	Verbosity int    `short:"v" long:"verbosity" description:"Verbosity of output (higher number = more output, default 2 -> notice, warnings and errors only)" default:"2"`
	LogFile   string `long:"log_file" description:"File to log to (in addition to stdout)"`

	CleanFlags struct {
		LowWaterMark   cli.ByteSize `short:"l" long:"low_water_mark" description:"Size of cache to clean down to" default:"18G"`
		HighWaterMark  cli.ByteSize `short:"i" long:"high_water_mark" description:"Max size of cache to clean at" default:"20G"`
		CleanFrequency cli.Duration `short:"f" long:"clean_frequency" description:"Frequency to clean cache at" default:"10m"`
		MaxArtifactAge cli.Duration `short:"m" long:"max_artifact_age" description:"Clean any artifact that's not been read in this long" default:"720h"`
	} `group:"Options controlling when to clean the cache"`

	TLSFlags struct {
		KeyFile       string `long:"key_file" description:"File containing PEM-encoded private key."`
		CertFile      string `long:"cert_file" description:"File containing PEM-encoded certificate"`
		CACertFile    string `long:"ca_cert_file" description:"File containing PEM-encoded CA certificate"`
		WritableCerts string `long:"writable_certs" description:"File or directory containing certificates that are allowed to write to the cache"`
		ReadonlyCerts string `long:"readonly_certs" description:"File or directory containing certificates that are allowed to read from the cache"`
	} `group:"Options controlling TLS communication & authentication"`

	ClusterFlags struct {
		ClusterPort      int    `long:"cluster_port" default:"7946" description:"Port to gossip among cluster nodes on"`
		ClusterAddresses string `short:"c" long:"cluster_addresses" description:"Comma-separated addresses of one or more nodes to join a cluster"`
		SeedCluster      bool   `long:"seed_cluster" description:"Seeds a new cache cluster."`
		ClusterSize      int    `long:"cluster_size" description:"Number of nodes to expect in the cluster.\nMust be passed if --seed_cluster is, has no effect otherwise."`
	} `group:"Options controlling clustering behaviour"`
}

func main() {
	cli.ParseFlagsOrDie("Please RPC cache server", "5.5.0", &opts)
	cli.InitLogging(opts.Verbosity)
	if opts.LogFile != "" {
		cli.InitFileLogging(opts.LogFile, opts.Verbosity)
	}
	if (opts.TLSFlags.KeyFile == "") != (opts.TLSFlags.CertFile == "") {
		log.Fatalf("Must pass both --key_file and --cert_file if you pass one")
	} else if opts.TLSFlags.KeyFile == "" && (opts.TLSFlags.WritableCerts != "" || opts.TLSFlags.ReadonlyCerts != "") {
		log.Fatalf("You can only use --writable_certs / --readonly_certs with https (--key_file and --cert_file)")
	}

	if opts.ClusterFlags.SeedCluster {
		if opts.ClusterFlags.ClusterSize < 2 {
			log.Fatalf("You must pass a cluster size of > 1 when initialising the seed node.")
		}
		server.InitCluster(opts.ClusterFlags.ClusterPort, opts.ClusterFlags.ClusterSize)
	} else if opts.ClusterFlags.ClusterAddresses != "" {
		server.JoinCluster(opts.ClusterFlags.ClusterPort, strings.Split(opts.ClusterFlags.ClusterAddresses, ","))
	}

	log.Notice("Scanning existing cache directory %s...", opts.Dir)
	cache := server.NewCache(opts.Dir, time.Duration(opts.CleanFlags.CleanFrequency),
		time.Duration(opts.CleanFlags.MaxArtifactAge),
		uint64(opts.CleanFlags.LowWaterMark), uint64(opts.CleanFlags.HighWaterMark))
	log.Notice("Starting up RPC cache server on port %d...", opts.Port)
	server.ServeGrpcForever(opts.Port, cache, opts.TLSFlags.KeyFile, opts.TLSFlags.CertFile,
		opts.TLSFlags.CACertFile, opts.TLSFlags.ReadonlyCerts, opts.TLSFlags.WritableCerts)
}
