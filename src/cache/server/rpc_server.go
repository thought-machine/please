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
	"path"
	"path/filepath"

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
	readonlyKeys map[string]*x509.Certificate
	writableKeys map[string]*x509.Certificate
}

func (r *RpcCacheServer) Store(ctx context.Context, req *pb.StoreRequest) (*pb.StoreResponse, error) {
	if err := r.authenticateClient(r.writableKeys, ctx); err != nil {
		return nil, err
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
	if err := r.authenticateClient(r.readonlyKeys, ctx); err != nil {
		return nil, err
	}
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
	if err := r.authenticateClient(r.writableKeys, ctx); err != nil {
		return nil, err
	}
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

func (r *RpcCacheServer) authenticateClient(certs map[string]*x509.Certificate, ctx context.Context) error {
	if len(certs) == 0 {
		return nil // Open to anyone.
	}
	p, ok := peer.FromContext(ctx)
	if !ok {
		return fmt.Errorf("Missing client certificate")
	}
	info, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return fmt.Errorf("Could not extract auth info")
	}
	if len(info.State.PeerCertificates) == 0 {
		return fmt.Errorf("No peer certificate available")
	}
	cert := info.State.PeerCertificates[0]
	if okCert := certs[string(cert.RawSubject)]; okCert != nil && okCert.Equal(cert) {
		return nil
	}
	return fmt.Errorf("Invalid or unknown certificate")
}

func loadKeys(filename string) map[string]*x509.Certificate {
	ret := map[string]*x509.Certificate{}
	if err := filepath.Walk(filename, func(name string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		} else if !info.IsDir() {
			data, err := ioutil.ReadFile(filename)
			if err != nil {
				log.Fatalf("Failed to read cert from %s: %s", filename, err)
			}
			p, _ := pem.Decode(data)
			if p == nil {
				log.Fatalf("Couldn't decode PEM data from %s: %s", filename, err)
			}
			cert, err := x509.ParseCertificate(p.Bytes)
			if err != nil {
				log.Fatalf("Couldn't parse certificate from %s: %s", filename, err)
			}
			ret[string(cert.RawSubject)] = cert
		}
		return nil
	}); err != nil {
		log.Fatalf("%s", err)
	}
	return ret
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
	pb.RegisterRpcCacheServer(s, r)
	healthserver := health.NewHealthServer()
	healthserver.SetServingStatus("plz-rpc-cache", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(s, healthserver)
	return s, lis
}

// ServeGrpcForever constructs a new server on the given port and serves until killed.
func ServeGrpcForever(port int, cache *Cache, keyFile, certFile, caCertFile, readonlyKeys, writableKeys string) {
	s, lis := BuildGrpcServer(port, cache, keyFile, certFile, caCertFile, readonlyKeys, writableKeys)
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
