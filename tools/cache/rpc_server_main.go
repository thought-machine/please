package main

import (
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"strings"
	"time"

	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/tools/cache/cluster"
	"github.com/thought-machine/please/tools/cache/server"
)

var log = logging.MustGetLogger("rpc_cache_server")

var opts struct {
	Usage     string        `usage:"rpc_cache_server is a server for Please's remote RPC cache.\n\nSee https://please.build/cache.html for more information."`
	Verbosity cli.Verbosity `short:"v" long:"verbosity" default:"notice" description:"Verbosity of output (higher number = more output)"`
	Port      int           `short:"p" long:"port" description:"Port to serve on" default:"7677"`
	HTTPPort  int           `short:"h" long:"http_port" description:"Port to serve HTTP on (for profiling, metrics etc)"`
	Dir       string        `short:"d" long:"dir" description:"Directory to write into" default:"plz-rpc-cache"`
	LogFile   string        `long:"log_file" description:"File to log to (in addition to stdout)"`

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
		NodeName         string `long:"node_name" env:"NODE_NAME" description:"Name of this node in the cluster. Only usually needs to be passed if running multiple nodes on the same machine, when it should be unique."`
		SeedIf           string `long:"seed_if" description:"Makes us the seed (overriding seed_cluster) if node_name matches this value and we can't resolve any cluster addresses. This makes it a lot easier to set up in automated deployments like Kubernetes."`
		AdvertiseAddr    string `long:"advertise_addr" env:"NODE_IP" description:"IP address to advertise to other cluster nodes"`
	} `group:"Options controlling clustering behaviour"`
}

// handleHTTP implements the http.Handler interface to return some simple stats.
func handleHTTP(w http.ResponseWriter, cache *server.Cache) {
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("X-Clacks-Overhead", "GNU Terry Pratchett")
	w.Write([]byte(fmt.Sprintf("Total size: %d bytes\nNum files: %d\n", cache.TotalSize(), cache.NumFiles())))
}

// serveHTTP serves HTTP responses (including metrics) on the given port.
func serveHTTP(port int, cache *server.Cache) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", prometheus.Handler())
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { handleHTTP(w, cache) })
	s := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
	go func() {
		if opts.TLSFlags.KeyFile != "" {
			log.Fatalf("%s\n", s.ListenAndServeTLS(opts.TLSFlags.CertFile, opts.TLSFlags.KeyFile))
		} else {
			log.Fatalf("%s\n", s.ListenAndServe())
		}
	}()
	log.Notice("Serving HTTP stats on port %d", port)
}

func main() {
	cli.ParseFlagsOrDie("Please RPC cache server", &opts)
	cli.InitLogging(opts.Verbosity)
	if opts.LogFile != "" {
		cli.InitFileLogging(opts.LogFile, opts.Verbosity)
	}
	if (opts.TLSFlags.KeyFile == "") != (opts.TLSFlags.CertFile == "") {
		log.Fatalf("Must pass both --key_file and --cert_file if you pass one")
	} else if opts.TLSFlags.KeyFile == "" && (opts.TLSFlags.WritableCerts != "" || opts.TLSFlags.ReadonlyCerts != "") {
		log.Fatalf("You can only use --writable_certs / --readonly_certs with https (--key_file and --cert_file)")
	}

	log.Notice("Scanning existing cache directory %s...", opts.Dir)
	cache := server.NewCache(opts.Dir, time.Duration(opts.CleanFlags.CleanFrequency),
		time.Duration(opts.CleanFlags.MaxArtifactAge),
		uint64(opts.CleanFlags.LowWaterMark), uint64(opts.CleanFlags.HighWaterMark))

	var clusta *cluster.Cluster
	if opts.ClusterFlags.SeedIf != "" && opts.ClusterFlags.SeedIf == opts.ClusterFlags.NodeName {
		ips, err := net.LookupIP(opts.ClusterFlags.ClusterAddresses)
		opts.ClusterFlags.SeedCluster = err != nil || len(ips) == 0
	}
	if opts.ClusterFlags.SeedCluster {
		if opts.ClusterFlags.ClusterSize < 2 {
			log.Fatalf("You must pass a cluster size of > 1 when initialising the seed node.")
		}
		clusta = cluster.NewCluster(opts.ClusterFlags.ClusterPort, opts.Port, opts.ClusterFlags.NodeName, opts.ClusterFlags.AdvertiseAddr)
		clusta.Init(opts.ClusterFlags.ClusterSize)
	} else if opts.ClusterFlags.ClusterAddresses != "" {
		clusta = cluster.NewCluster(opts.ClusterFlags.ClusterPort, opts.Port, opts.ClusterFlags.NodeName, opts.ClusterFlags.AdvertiseAddr)
		clusta.Join(strings.Split(opts.ClusterFlags.ClusterAddresses, ","))
	}

	log.Notice("Starting up RPC cache server on port %d...", opts.Port)
	s, lis := server.BuildGrpcServer(opts.Port, cache, clusta, opts.TLSFlags.KeyFile, opts.TLSFlags.CertFile,
		opts.TLSFlags.CACertFile, opts.TLSFlags.ReadonlyCerts, opts.TLSFlags.WritableCerts)

	grpc_prometheus.Register(s)
	grpc_prometheus.EnableHandlingTimeHistogram()
	if opts.HTTPPort != 0 {
		go serveHTTP(opts.HTTPPort, cache)
	}
	server.ServeGrpcForever(s, lis)
}
