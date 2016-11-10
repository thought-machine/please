package cluster

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "cache/proto/rpc_cache"
)

func TestBringUpCluster(t *testing.T) {
	c1 := newCluster(5995, "c1")
	m1 := newRPCServer(c1, 6995)
	c1.Init(3)
	log.Notice("Cluster seeded")

	c2 := newCluster(5996, "c2")
	m2 := newRPCServer(c2, 6996)
	c2.Join([]string{"localhost:5995"})
	log.Notice("c2 joined cluster")

	expected := []*pb.Node{
		&pb.Node{},
	}
	// Both nodes should agree about the member list
	assert.Equal(t, expected, c1.GetMembers())
	assert.Equal(t, expected, c2.GetMembers())

	c3 := newCluster(5997, "c3")
	m3 := newRPCServer(c2, 6997)
	c3.Join([]string{"localhost:5995", "localhost:5996"})

	assert.Equal(t, 0, m1.Replications)
	assert.Equal(t, 0, m2.Replications)
	assert.Equal(t, 0, m3.Replications)
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

// newRPCServer creates a new mockRPCServer, starts a gRPC server running it, and returns it.
// It's not possible to stop it again...
func newRPCServer(cluster *Cluster, port int) *mockRPCServer {
	m := &mockRPCServer{cluster: cluster}
	s := grpc.NewServer()
	pb.RegisterRpcServerServer(s, m)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", port, err)
	}
	go s.Serve(lis)
	return m
}
