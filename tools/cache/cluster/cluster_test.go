package cluster

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "github.com/thought-machine/please/src/cache/proto/rpc_cache"
	"github.com/thought-machine/please/src/cache/tools"
)

func TestBringUpCluster(t *testing.T) {
	lis := openRPCPort(6995)
	c1 := NewCluster(5995, 6995, "c1", "")
	m1 := newRPCServer(c1, lis)
	c1.Init(3)
	log.Notice("Cluster seeded")

	lis = openRPCPort(6996)
	c2 := NewCluster(5996, 6996, "c2", "")
	m2 := newRPCServer(c2, lis)
	c2.Join([]string{"127.0.0.1:5995"})
	log.Notice("c2 joined cluster")

	// We want to get the address of the nodes; on a typical machine there are multiple
	// available interfaces and memberlist will intelligently choose one. We don't want to
	// try to second-guess their logic here so we sneakily grab whatever it thinks the
	// local node's address is.
	addr := c1.list.LocalNode().Addr.String()

	expected := []*pb.Node{
		{
			Name:      "c1",
			Address:   addr + ":6995",
			HashBegin: tools.HashPoint(0, 3),
			HashEnd:   tools.HashPoint(1, 3),
		},
		{
			Name:      "c2",
			Address:   addr + ":6996",
			HashBegin: tools.HashPoint(1, 3),
			HashEnd:   tools.HashPoint(2, 3),
		},
	}
	// Both nodes should agree about the member list
	assert.Equal(t, expected, c1.GetMembers())
	assert.Equal(t, expected, c2.GetMembers())

	lis = openRPCPort(6997)
	c3 := NewCluster(5997, 6997, "c3", "")
	m3 := newRPCServer(c2, lis)
	c3.Join([]string{"127.0.0.1:5995", "127.0.0.1:5996"})

	expected = []*pb.Node{
		{
			Name:      "c1",
			Address:   addr + ":6995",
			HashBegin: tools.HashPoint(0, 3),
			HashEnd:   tools.HashPoint(1, 3),
		},
		{
			Name:      "c2",
			Address:   addr + ":6996",
			HashBegin: tools.HashPoint(1, 3),
			HashEnd:   tools.HashPoint(2, 3),
		},
		{
			Name:      "c3",
			Address:   addr + ":6997",
			HashBegin: tools.HashPoint(2, 3),
			HashEnd:   tools.HashPoint(3, 3),
		},
	}

	// All three nodes should agree about the member list
	assert.Equal(t, expected, c1.GetMembers())
	assert.Equal(t, expected, c2.GetMembers())
	assert.Equal(t, expected, c3.GetMembers())

	assert.Equal(t, 0, m1.Replications)
	assert.Equal(t, 0, m2.Replications)
	assert.Equal(t, 0, m3.Replications)

	// Now test replications.
	c1.ReplicateArtifacts(&pb.StoreRequest{
		Hash: []byte{0, 0, 0, 0},
	})
	// This replicates onto node 2 because that's got the relevant bit of the hash space.
	assert.Equal(t, 0, m1.Replications)
	assert.Equal(t, 1, m2.Replications)
	assert.Equal(t, 0, m3.Replications)

	// The same request going to node 2 should replicate it onto node 1.
	c2.ReplicateArtifacts(&pb.StoreRequest{
		Hash: []byte{0, 0, 0, 0},
	})
	assert.Equal(t, 1, m1.Replications)
	assert.Equal(t, 1, m2.Replications)
	assert.Equal(t, 0, m3.Replications)

	// Delete requests should get replicated around the whole cluster (because they delete
	// all hashes of an artifact, and so those could be anywhere).
	c1.DeleteArtifacts(&pb.DeleteRequest{})
	assert.Equal(t, 1, m1.Replications)
	assert.Equal(t, 2, m2.Replications)
	assert.Equal(t, 1, m3.Replications)
	c2.DeleteArtifacts(&pb.DeleteRequest{})
	assert.Equal(t, 2, m1.Replications)
	assert.Equal(t, 2, m2.Replications)
	assert.Equal(t, 2, m3.Replications)
	c3.DeleteArtifacts(&pb.DeleteRequest{})
	assert.Equal(t, 3, m1.Replications)
	assert.Equal(t, 3, m2.Replications)
	assert.Equal(t, 2, m3.Replications)
}

// mockRPCServer is a fake RPC server we use for this test.
type mockRPCServer struct {
	cluster      *Cluster
	Replications int
}

func (r *mockRPCServer) Join(ctx context.Context, req *pb.JoinRequest) (*pb.JoinResponse, error) {
	return r.cluster.AddNode(req), nil
}

func (r *mockRPCServer) Replicate(ctx context.Context, req *pb.ReplicateRequest) (*pb.ReplicateResponse, error) {
	r.Replications++
	return &pb.ReplicateResponse{Success: true}, nil
}

// openRPCPort opens a port for the gRPC server.
// This is rather awkwardly split up from below to try to avoid races around the port opening.
// There's something of a circular dependency between starting the gossip service (which triggers
// RPC calls) and starting the gRPC server (which refers to said gossip service).
func openRPCPort(port int) net.Listener {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", port, err)
	}
	return lis
}

// newRPCServer creates a new mockRPCServer, starts a gRPC server running it, and returns it.
// It's not possible to stop it again...
func newRPCServer(cluster *Cluster, lis net.Listener) *mockRPCServer {
	m := &mockRPCServer{cluster: cluster}
	s := grpc.NewServer()
	pb.RegisterRpcServerServer(s, m)
	go s.Serve(lis)
	return m
}
