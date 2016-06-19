// Test only at this point to make sure we can build grpc correctly.
// Later this will turn into a proper RPC cache server implementation.
package server

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net"
	"path"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/peer"

	pb "cache/proto/rpc_cache"
)

type RpcCacheServer struct {
	cache        *Cache
	readonlyKeys map[[]byte]bool
	writableKeys map[[]byte]bool
}

func (r *RpcCacheServer) Store(ctx context.Context, req *pb.StoreRequest) (*pb.StoreResponse, error) {
	if err := r.authenticateClient(r.readonlyKeys, ctx); err != nil {
		return err
	}
	arch := req.Os + "_" + req.Arch
	hash := base64.RawURLEncoding.EncodeToString(req.Hash)
	for _, artifact := range req.Artifacts {
		path := path.Join(arch, artifact.Package, artifact.Target, hash, artifact.File)
		if err := r.cache.StoreArtifact(path, artifact.Body); err != nil {
			return &pb.StoreResponse{Success: false}, nil
		}
	}
	return &pb.StoreResponse{Success: true}, nil
}

func (r *RpcCacheServer) Retrieve(ctx context.Context, req *pb.RetrieveRequest) (*pb.RetrieveResponse, error) {
	response := pb.RetrieveResponse{Success: true}
	arch := req.Os + "_" + req.Arch
	hash := base64.RawURLEncoding.EncodeToString(req.Hash)
	for _, artifact := range req.Artifacts {
		root := path.Join(arch, artifact.Package, artifact.Target, hash)
		fileRoot := path.Join(root, artifact.File)
		art, err := r.cache.RetrieveArtifact(fileRoot)
		if err != nil {
			log.Debug("Failed to retrieve artifact %s: %s", fileRoot, err)
			return &pb.RetrieveResponse{Success: false}, nil
		}
		for name, body := range art {
			response.Artifacts = append(response.Artifacts, &pb.Artifact{
				Package: artifact.Package,
				Target:  artifact.Target,
				File:    name[len(root)+1 : len(name)],
				Body:    body,
			})
		}
	}
	return &response, nil
}

func (r *RpcCacheServer) Delete(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	if req.Everything {
		return &pb.DeleteResponse{Success: r.cache.DeleteAllArtifacts() == nil}, nil
	}
	success := true
	arch := req.Os + "_" + req.Arch
	for _, artifact := range req.Artifacts {
		if r.cache.DeleteArtifact(path.Join(arch, artifact.Package, artifact.Target)) != nil {
			success = false
		}
	}
	return &pb.DeleteResponse{Success: success}, nil
}

func (r *RpcCacheServer) authenticateClient(keys map[[]byte]bool, ctx context.Context) error {
	if len(keys) == 0 {
		return nil // Open to anyone.
	}
	_, ok := peer.FromContext(ctx)
	if !ok {
		return fmt.Errorf("Missing client certificate")
	}
	return nil
}

// BuildGrpcServer creates a new, unstarted grpc.Server and returns it.
// It also returns a net.Listener to start it on.
func BuildGrpcServer(port int, cache *Cache, keyFile, certFile, caCertFile, readonlyKeys, writableKeys string) (*grpc.Server, net.Listener) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", port, err)
	}
	s := serverWithAuth(keyFile, certFile, caCertFile)
	r := &RpcCacheServer{cache: cache}
	if readonlyKeys != "" {
		//		r.readonlyKeys =
	}
	pb.RegisterRpcCacheServer(s, r)
	healthserver := health.NewHealthServer()
	healthserver.SetServingStatus("plz-rpc-cache", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(s, healthserver)
	return s, lis
}

// ServeGrpcForever constructs a new server on the given port and serves until killed.
func ServeGrpcForever(port int, cache *Cache, keyFile, certFile, caCertFile string) {
	s, lis := BuildGrpcServer(port, cache, keyFile, certFile, caCertFile)
	log.Notice("Serving RPC cache on port %d", port)
	s.Serve(lis)
}

// serverWithAuth builds a gRPC server, possibly with authentication if key / cert files are given.
func serverWithAuth(keyFile, certFile, caCertFile string) *grpc.Server {
	if keyFile == "" {
		return grpc.NewServer() // No auth.
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
	return grpc.NewServer(grpc.Creds(credentials.NewTLS(&config)))
}
