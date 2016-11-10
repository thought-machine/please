// Implements the RPC server for cache communication.

package cluster

import (
	"encoding/base64"
	"path"

	"github.com/hashicorp/memberlist"

	pb "cache/proto/rpc_cache"
)

// RPCServer implements the gRPC server for communication between cache nodes.
type RPCServer struct {
	cache   *Cache
	cluster *Cluster
}

func (r *RPCServer) Join(ctx context.Context, req *pb.JoinRequest) (*pb.JoinResponse, error) {
	// TODO(pebers): Authentication.
	node := r.cluster.newNode(&memberlist.Node{
		Name: req.Name,
		Addr: req.Address,
	})
	if node == nil {
		return &pb.JoinResponse{Success: false}, nil
	}
	return &pb.JoinResponse{
		Success:   true,
		HashBegin: node.HashBegin,
		HashEnd:   node.HashEnd,
		Nodes:     cluster.GetMembers(),
	}, nil
}

func (r *RPCServer) Replicate(ctx context.Context, req *pb.ReplicateRequest) (*pb.ReplicateResponse, error) {
	// TODO(pebers): the code here is very similar to that in rpc_server.go, share it somehow.
	arch := req.Os + "_" + req.Arch
	hash := base64.RawURLEncoding.EncodeToString(req.Hash)
	for _, artifact := range req.Artifacts {
		path := path.Join(arch, artifact.Package, artifact.Target, hash, artifact.File)
		if err := r.cache.StoreArtifact(path, artifact.Body); err != nil {
			return &pb.ReplicateResponse{Success: false}, nil
		}
	}
	return &pb.ReplicateResponse{Success: true}, nil
}

// ReplicateArtifacts replicates artifacts from this node to another.
func ReplicateArtifacts(cluster *Cluster, req *pb.StoreRequest) {
	address := cluster.getAlternateNode(req.Hash)
	if address == "" {
		log.Warning("Couldn't get alternate address, will not replicate artifact")
		return
	}
	client, err := cluster.getRPCClient(address)
	if err != nil {
		log.Error("Failed to get RPC client for %s: %s", address, err)
		return
	}
	if resp, err := client.Replicate(&pb.ReplicateRequest{
		Artifacts: req.Artifacts,
		Os:        req.Os,
		Arch:      req.Arch,
		Hash:      req.Hash,
	}); err != nil {
		log.Error("Error replicating artifact: %s", err)
	} else if !resp.Success {
		log.Error("Failed to replicate artifact to %s", address)
	}
}
