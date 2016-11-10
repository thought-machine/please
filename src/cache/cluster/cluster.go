// Package cluster contains functions for dealing with a cluster of plz cache nodes.
//
// Clustering the cache provides redundancy and increased performance
// for large caches. Right now the functionality is a little limited,
// there's no online rehashing and the replication factor is fixed at 2.
//
// The general approach here errs heavily on the side of simplicity and
// less on zero-downtime reliability since, at the end of the day, this
// is only a cache server. It would be a good idea not to be too mean to
// it by, for example, having many nodes join and leave the cluster rapidly
// under different names.
package cluster

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/hashicorp/memberlist"
	"google.golang.org/grpc"
	"gopkg.in/op/go-logging.v1"

	pb "cache/proto/rpc_cache"
	"cache/tools"
)

var log = logging.MustGetLogger("cluster")

// A Cluster handles communication between a set of clustered cache servers.
type Cluster struct {
	list *memberlist.Memberlist
	// nodes is a list of nodes that is initialised by the original seed
	// and replicated between any other nodes that join after.
	nodes []*pb.Node

	// clients is a pool of gRPC clients to the other cluster nodes.
	clients map[string]pb.RpcServerClient
	// clientMutex protects concurrent access to clients.
	clientMutex sync.RWMutex

	// hashStart and hashEnd are the endpoints in the hash space for this client.
	hashStart uint32
	hashEnd   uint32
}

// NewCluster creates a new Cluster object and starts listening on the given port.
func NewCluster(port int) *Cluster {
	return newCluster(port, "")
}

// newCluster is split from the above for testing purposes.
func newCluster(port int, name string) *Cluster {
	c := memberlist.DefaultLANConfig()
	c.BindPort = port
	c.AdvertisePort = port
	if name != "" {
		c.Name = name
	}
	list, err := memberlist.Create(c)
	if err != nil {
		log.Fatalf("Failed to create new memberlist: %s", err)
	}
	n := list.LocalNode()
	log.Notice("Memberlist initialised, this node is %s / %s", n.Name, n.Addr)
	return &Cluster{
		list:    list,
		clients: map[string]pb.RpcServerClient{},
	}
}

// JoinCluster joins an existing plz cache cluster.
func (cluster *Cluster) Join(members []string) {
	// Talk to the other nodes to request to join.
	if _, err := cluster.list.Join(members); err != nil {
		log.Fatalf("Failed to join cluster: %s", err)
	}
	for _, node := range cluster.GetMembers() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if client, err := cluster.getRPCClient(node.Address); err != nil {
			log.Error("Error getting RPC client for %s: %s", node.Address, err)
		} else if resp, err := client.Join(ctx, &pb.JoinRequest{
			Name:    cluster.list.LocalNode().Name,
			Address: cluster.list.LocalNode().Addr.String(),
		}); err != nil {
			log.Error("Error communicating with %s: %s", node.Address, err)
		} else if !resp.Success {
			log.Fatalf("We have not been allowed to join the cluster :(")
		} else {
			cluster.hashStart = resp.HashBegin
			cluster.hashEnd = resp.HashEnd
			cluster.nodes = resp.Nodes
			return
		}
	}
	log.Fatalf("Unable to contact any other cluster members")
}

// InitCluster seeds a new plz cache cluster.
func (cluster *Cluster) Init(size int) {
	// Create the node list
	cluster.nodes = make([]*pb.Node, size)
	// We're node 0
	cluster.newNode(cluster.list.LocalNode())
	// And there aren't any others yet, so we're done.
}

// GetMembers returns the set of currently known cache members.
func (cluster *Cluster) GetMembers() []*pb.Node {
	// TODO(pebers): this is quadratic so would be bad on large clusters.
	// We might also want to refresh the members in the background if this proves slow?
	for _, m := range cluster.list.Members() {
		cluster.newNode(m)
	}
	return cluster.nodes
}

// newNode constructs one of our canonical nodes from a memberlist.Node.
// This includes allocating it hash space.
func (cluster *Cluster) newNode(node *memberlist.Node) *pb.Node {
	for i, n := range cluster.nodes {
		if n == nil || n.Name == "" || n.Name == node.Name {
			// Available slot. Or, if they identified as an existing node, they can take that space over.
			if n != nil && n.Name == node.Name {
				log.Warning("Node %s / %s is taking over slot %d", node.Name, node.Addr, i)
			} else {
				log.Notice("Populating node %d: %s / %s", i, node.Name, node.Addr)
			}
			cluster.nodes[i] = &pb.Node{
				Name:      node.Name,
				Address:   node.Addr.String(),
				HashBegin: tools.HashPoint(i, len(cluster.nodes)),
				HashEnd:   tools.HashPoint(i+1, len(cluster.nodes)),
			}
			return cluster.nodes[i]
		}
	}
	log.Warning("Node %s / %s attempted to join, but there is no space available.", node.Name, node.Addr)
	return nil
}

// getRPCClient returns an RPC client for the given server.
func (cluster *Cluster) getRPCClient(address string) (pb.RpcServerClient, error) {
	cluster.clientMutex.RLock()
	client, present := cluster.clients[address]
	cluster.clientMutex.RUnlock()
	if present {
		return client, nil
	}
	// TODO(pebers): add credentials.
	connection, err := grpc.Dial(address, grpc.WithTimeout(5*time.Second), grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	client = pb.NewRpcServerClient(connection)
	cluster.clientMutex.Lock()
	cluster.clients[address] = client
	cluster.clientMutex.Unlock()
	return client, nil
}

// getAddress returns the replica node for the given hash (i.e. whichever one is not us,
// we don't really know for sure when calling this if we are the primary or not).
func (cluster *Cluster) getAlternateNode(hash []byte) string {
	point := tools.Hash(hash)
	if point >= cluster.hashStart && point < cluster.hashEnd {
		// We've got this point, use the alternate.
		point = tools.AlternateHash(hash)
	}
	for _, n := range cluster.nodes {
		if point >= n.HashBegin && point < n.HashEnd {
			return n.Address
		}
	}
	log.Warning("No cluster node found for hash point %d", point)
	return ""
}

// ReplicateArtifacts replicates artifacts from this node to another.
func (cluster *Cluster) ReplicateArtifacts(req *pb.StoreRequest) {
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if resp, err := client.Replicate(ctx, &pb.ReplicateRequest{
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

// AddNode adds a new node that's applying to join the cluster.
func (cluster *Cluster) AddNode(req *pb.JoinRequest) *pb.JoinResponse {
	node := cluster.newNode(&memberlist.Node{
		Name: req.Name,
		Addr: net.ParseIP(req.Address),
	})
	if node == nil {
		return &pb.JoinResponse{Success: false}
	}
	return &pb.JoinResponse{
		Success:   true,
		HashBegin: node.HashBegin,
		HashEnd:   node.HashEnd,
		Nodes:     cluster.GetMembers(),
	}
}
