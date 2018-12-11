package server

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"

	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	_ "google.golang.org/grpc/encoding/gzip" // Registers the gzip compressor at init
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	pb "github.com/thought-machine/please/src/cache/proto/rpc_cache"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/tools/cache/cluster"
)

// maxMsgSize is the maximum message size our gRPC server accepts.
// We deliberately set this to something high since we don't want to limit artifact size here.
const maxMsgSize = 200 * 1024 * 1024

// metricsOnce is used to track whether or not we've registered the server metrics.
// In normal operation this only happens once but in tests it can happen multiple times.
var metricsOnce sync.Once

func init() {
	// When tracing is enabled, it appears to keep references to messages alive, possibly indefinitely (?).
	// This is very bad for us since our messages are large, it can result in leaking memory very quickly
	// and ultimately OOM errors. Disabling tracing appears to alleviate the problem.
	grpc.EnableTracing = false
}

// A RPCCacheServer implements our RPC cache, including communication in a cluster.
type RPCCacheServer struct {
	cache                                                                          *Cache
	readonlyKeys                                                                   map[string]*x509.Certificate
	writableKeys                                                                   map[string]*x509.Certificate
	cluster                                                                        *cluster.Cluster
	retrievedCounter, storedCounter, retrievedBytes, storedBytes, retrieveFailures *prometheus.CounterVec
}

// Store implements the Store RPC to store an artifact in the cache.
func (r *RPCCacheServer) Store(ctx context.Context, req *pb.StoreRequest) (*pb.StoreResponse, error) {
	if err := r.authenticateClient(ctx, r.writableKeys); err != nil {
		return nil, err
	}
	success := storeArtifact(r.cache, req.Os, req.Arch, req.Hash, req.Artifacts, req.Hostname, extractAddress(ctx), "")
	if success && r.cluster != nil {
		// Replicate this artifact to another node. Doesn't have to be done synchronously.
		go r.cluster.ReplicateArtifacts(req)
	}
	if success {
		r.storedCounter.WithLabelValues(req.Arch).Inc()
		total := 0
		for _, artifact := range req.Artifacts {
			total += len(artifact.Body)
		}
		r.storedBytes.WithLabelValues(req.Arch).Add(float64(total))
	}
	return &pb.StoreResponse{Success: success}, nil
}

// storeArtifact stores a series of artifacts in the cache.
// Broken out of above to share with Replicate below.
func storeArtifact(cache *Cache, os, arch string, hash []byte, artifacts []*pb.Artifact, hostname, address, peer string) bool {
	arch = os + "_" + arch
	hashStr := base64.RawURLEncoding.EncodeToString(hash)
	for _, artifact := range artifacts {
		dir := path.Join(arch, artifact.Package, artifact.Target, hashStr)
		file := path.Join(dir, artifact.File)
		if err := cache.StoreArtifact(file, artifact.Body, artifact.Symlink); err != nil {
			return false
		}
		go cache.StoreMetadata(dir, hostname, address, peer)
	}
	return true
}

// Retrieve implements the Retrieve RPC to retrieve artifacts from the cache.
func (r *RPCCacheServer) Retrieve(ctx context.Context, req *pb.RetrieveRequest) (*pb.RetrieveResponse, error) {
	if err := r.authenticateClient(ctx, r.readonlyKeys); err != nil {
		return nil, err
	}
	response := pb.RetrieveResponse{Success: true}
	arch := req.Os + "_" + req.Arch
	hash := base64.RawURLEncoding.EncodeToString(req.Hash)
	total := 0
	for _, artifact := range req.Artifacts {
		root := path.Join(arch, artifact.Package, artifact.Target, hash)
		fileRoot := path.Join(root, artifact.File)
		arts, err := r.cache.RetrieveArtifact(fileRoot)
		if err != nil {
			log.Debug("Failed to retrieve artifact %s: %s", fileRoot, err)
			r.retrieveFailures.WithLabelValues(req.Arch).Inc()
			return &pb.RetrieveResponse{Success: false}, nil
		}
		for _, art := range arts {
			response.Artifacts = append(response.Artifacts, &pb.Artifact{
				Package: artifact.Package,
				Target:  artifact.Target,
				File:    art.File[len(root)+1:],
				Body:    art.Body,
				Symlink: art.Symlink,
			})
			total += len(art.Body)
		}
	}
	r.retrievedCounter.WithLabelValues(req.Arch).Inc()
	r.retrievedBytes.WithLabelValues(req.Arch).Add(float64(total))
	return &response, nil
}

// Delete implements the Delete RPC to delete an artifact from the cache.
func (r *RPCCacheServer) Delete(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	if err := r.authenticateClient(ctx, r.writableKeys); err != nil {
		return nil, err
	}
	if req.Everything {
		return &pb.DeleteResponse{Success: r.cache.DeleteAllArtifacts() == nil}, nil
	}
	success := deleteArtifact(r.cache, req.Os, req.Arch, req.Artifacts)
	if success && r.cluster != nil {
		// Delete this artifact from other nodes. Doesn't have to be done synchronously.
		go r.cluster.DeleteArtifacts(req)
	}
	return &pb.DeleteResponse{Success: success}, nil
}

// deleteArtifact handles the actual removal of artifacts from the cache.
// It's split out from Delete to share with replication RPCs below.
func deleteArtifact(cache *Cache, os, arch string, artifacts []*pb.Artifact) bool {
	success := true
	for _, artifact := range artifacts {
		if cache.DeleteArtifact(path.Join(os+"_"+arch, artifact.Package, artifact.Target)) != nil {
			success = false
		}
	}
	return success
}

// ListNodes implements the RPC for clustered servers.
func (r *RPCCacheServer) ListNodes(ctx context.Context, req *pb.ListRequest) (*pb.ListResponse, error) {
	if err := r.authenticateClient(ctx, r.readonlyKeys); err != nil {
		return nil, err
	}
	if r.cluster == nil {
		return &pb.ListResponse{}, nil
	}
	return &pb.ListResponse{Nodes: r.cluster.GetMembers()}, nil
}

func (r *RPCCacheServer) authenticateClient(ctx context.Context, certs map[string]*x509.Certificate) error {
	if len(certs) == 0 {
		return nil // Open to anyone.
	}
	p, ok := peer.FromContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "Missing client certificate")
	}
	info, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return status.Error(codes.Unauthenticated, "Could not extract auth info")
	}
	if len(info.State.PeerCertificates) == 0 {
		return status.Error(codes.Unauthenticated, "No peer certificate available")
	}
	cert := info.State.PeerCertificates[0]
	okCert := certs[string(cert.RawSubject)]
	if okCert == nil || !okCert.Equal(cert) {
		return status.Error(codes.Unauthenticated, "Invalid or unknown certificate")
	}
	return nil
}

func extractAddress(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return ""
	}
	return p.Addr.String()
}

func loadKeys(filename string) map[string]*x509.Certificate {
	ret := map[string]*x509.Certificate{}
	if err := fs.Walk(filename, func(name string, isDir bool) error {
		if !isDir {
			data, err := ioutil.ReadFile(name)
			if err != nil {
				log.Fatalf("Failed to read cert from %s: %s", name, err)
			}
			p, _ := pem.Decode(data)
			if p == nil {
				log.Fatalf("Couldn't decode PEM data from %s: %s", name, err)
			}
			cert, err := x509.ParseCertificate(p.Bytes)
			if err != nil {
				log.Fatalf("Couldn't parse certificate from %s: %s", name, err)
			}
			ret[string(cert.RawSubject)] = cert
		}
		return nil
	}); err != nil {
		log.Fatalf("%s", err)
	}
	return ret
}

// RPCServer implements the gRPC server for communication between cache nodes.
type RPCServer struct {
	cache   *Cache
	cluster *cluster.Cluster
}

// Join implements the Join RPC for a new server joining the cluster.
func (r *RPCServer) Join(ctx context.Context, req *pb.JoinRequest) (*pb.JoinResponse, error) {
	// TODO(pebers): Authentication.
	return r.cluster.AddNode(req), nil
}

// Replicate implements the Replicate RPC for replicating an artifact from another node.
func (r *RPCServer) Replicate(ctx context.Context, req *pb.ReplicateRequest) (*pb.ReplicateResponse, error) {
	// TODO(pebers): Authentication.
	if req.Delete {
		return &pb.ReplicateResponse{
			Success: deleteArtifact(r.cache, req.Os, req.Arch, req.Artifacts),
		}, nil
	}
	return &pb.ReplicateResponse{
		Success: storeArtifact(r.cache, req.Os, req.Arch, req.Hash, req.Artifacts, req.Hostname, extractAddress(ctx), req.Peer),
	}, nil
}

// BuildGrpcServer creates a new, unstarted grpc.Server and returns it.
// It also returns a net.Listener to start it on.
func BuildGrpcServer(port int, cache *Cache, cluster *cluster.Cluster, keyFile, certFile, caCertFile, readonlyKeys, writableKeys string) (*grpc.Server, net.Listener) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", port, err)
	}
	s := serverWithAuth(keyFile, certFile, caCertFile)
	r := &RPCCacheServer{
		cache:   cache,
		cluster: cluster,
		retrievedCounter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "retrieved_count",
			Help: "Number of artifacts successfully retrieved",
		}, []string{"arch"}),
		storedCounter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "stored_count",
			Help: "Number of artifacts successfully stored",
		}, []string{"arch"}),
		retrievedBytes: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "retrieved_bytes",
			Help: "Number of bytes successfully retrieved",
		}, []string{"arch"}),
		storedBytes: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "stored_bytes",
			Help: "Number of bytes successfully stored",
		}, []string{"arch"}),
		retrieveFailures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "retrieve_failures",
			Help: "Number of failed retrieval attempts",
		}, []string{"arch"}),
	}
	if writableKeys != "" {
		r.writableKeys = loadKeys(writableKeys)
	}
	if readonlyKeys != "" {
		r.readonlyKeys = loadKeys(readonlyKeys)
		if len(r.readonlyKeys) > 0 {
			// This saves duplication when checking later; writable keys are implicitly readable too.
			for k, v := range r.writableKeys {
				if _, present := r.readonlyKeys[k]; !present {
					r.readonlyKeys[k] = v
				}
			}
		}
	}
	r2 := &RPCServer{cache: cache, cluster: cluster}
	pb.RegisterRpcCacheServer(s, r)
	pb.RegisterRpcServerServer(s, r2)
	healthserver := health.NewServer()
	healthserver.SetServingStatus("plz-rpc-cache", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(s, healthserver)
	metricsOnce.Do(func() {
		prometheus.MustRegister(r.retrievedCounter)
		prometheus.MustRegister(r.storedCounter)
		prometheus.MustRegister(r.retrievedBytes)
		prometheus.MustRegister(r.storedBytes)
		prometheus.MustRegister(r.retrieveFailures)
	})
	return s, lis
}

// ServeGrpcForever serves gRPC until killed using the given server.
// It's very simple and provided as a convenience so callers don't have to import grpc themselves.
func ServeGrpcForever(server *grpc.Server, lis net.Listener) {
	log.Notice("Serving RPC cache on %s", lis.Addr())
	go handleSignals(server)
	server.Serve(lis)
}

// serverWithAuth builds a gRPC server, possibly with authentication if key / cert files are given.
func serverWithAuth(keyFile, certFile, caCertFile string) *grpc.Server {
	if keyFile == "" {
		return grpc.NewServer(grpc.MaxRecvMsgSize(maxMsgSize), grpc.MaxSendMsgSize(maxMsgSize)) // No auth.
	}
	log.Debug("Loading x509 key pair from key: %s cert: %s", keyFile, certFile)
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatalf("Failed to load x509 key pair: %s", err)
	}
	config := tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequestClientCert,
	}
	if caCertFile != "" {
		cert, err := ioutil.ReadFile(caCertFile)
		if err != nil {
			log.Fatalf("Failed to read CA cert file: %s", err)
		}
		config.ClientCAs = x509.NewCertPool()
		if !config.ClientCAs.AppendCertsFromPEM(cert) {
			log.Fatalf("Failed to find any PEM certificates in CA cert")
		}
	}
	return grpc.NewServer(
		grpc.Creds(credentials.NewTLS(&config)),
		grpc.MaxRecvMsgSize(maxMsgSize),
		grpc.MaxSendMsgSize(maxMsgSize),
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
	)
}

// handleSignals received SIGTERM / SIGINT etc to gracefully shut down a gRPC server.
// Repeated signals cause the server to terminate at increasing levels of urgency.
func handleSignals(s *grpc.Server) {
	c := make(chan os.Signal, 3) // Channel should be buffered a bit
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	sig := <-c
	log.Warning("Received signal %s, gracefully shutting down gRPC server", sig)
	go s.GracefulStop()
	sig = <-c
	log.Warning("Received signal %s, non-gracefully shutting down gRPC server", sig)
	go s.Stop()
	sig = <-c
	log.Fatalf("Received signal %s, terminating\n", sig)
}
