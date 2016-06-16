// Test only at this point to make sure we can build grpc correctly.
// Later this will turn into a proper RPC cache server implementation.
package server

import (
	"encoding/base64"
	"fmt"
	"net"
	"path"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	pb "cache/proto/rpc_cache"
)

type RpcCacheServer struct {
	cache *Cache
}

func (r *RpcCacheServer) Store(ctx context.Context, req *pb.StoreRequest) (*pb.StoreResponse, error) {
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

// BuildGrpcServer creates a new, unstarted grpc.Server and returns it.
// It also returns a net.Listener to start it on.
func BuildGrpcServer(port int, cache *Cache) (*grpc.Server, net.Listener) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", port, err)
	}
	s := grpc.NewServer()
	r := &RpcCacheServer{cache: cache}
	pb.RegisterRpcCacheServer(s, r)
	healthserver := health.NewHealthServer()
	healthserver.SetServingStatus("plz-rpc-cache", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(s, healthserver)
	return s, lis
}

// ServeGrpcForever constructs a new server on the given port and serves until killed.
func ServeGrpcForever(port int, cache *Cache) {
	s, lis := BuildGrpcServer(port, cache)
	log.Notice("Serving RPC cache on port %d", port)
	s.Serve(lis)
}
