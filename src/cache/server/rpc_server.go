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

type RpcCacheServer struct{}

func (*RpcCacheServer) Store(ctx context.Context, req *pb.StoreRequest) (*pb.StoreResponse, error) {
	arch := req.Os + "_" + req.Arch
	hash := base64.RawURLEncoding.EncodeToString(req.Hash)
	for _, artifact := range req.Artifacts {
		path := path.Join(arch, artifact.Package, artifact.Target, hash, artifact.File)
		if err := StoreArtifact(path, artifact.Body); err != nil {
			return &pb.StoreResponse{Success: false}, nil
		}
	}
	return &pb.StoreResponse{Success: true}, nil
}

func (*RpcCacheServer) Retrieve(ctx context.Context, req *pb.RetrieveRequest) (*pb.RetrieveResponse, error) {
	response := pb.RetrieveResponse{Success: true}
	arch := req.Os + "_" + req.Arch
	hash := base64.RawURLEncoding.EncodeToString(req.Hash)
	for _, artifact := range req.Artifacts {
		root := path.Join(arch, artifact.Package, artifact.Target, hash)
		fileRoot := path.Join(root, artifact.File)
		art, err := RetrieveArtifact(fileRoot)
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

func (*RpcCacheServer) Delete(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	if req.Everything {
		return &pb.DeleteResponse{Success: DeleteAllArtifacts() == nil}, nil
	}
	success := true
	arch := req.Os + "_" + req.Arch
	for _, artifact := range req.Artifacts {
		success = success && (DeleteArtifact(path.Join(arch, artifact.Package, artifact.Target)) == nil)
	}
	return &pb.DeleteResponse{Success: success}, nil
}

func ServeGrpcForever(port int) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", port, err)
	}
	s := grpc.NewServer()
	pb.RegisterRpcCacheServer(s, &RpcCacheServer{})
	healthserver := health.NewHealthServer()
	healthserver.SetServingStatus("plz-rpc-cache", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(s, healthserver)
	log.Notice("Serving RPC cache on port %d", port)
	s.Serve(lis)
}
