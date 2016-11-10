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
	"sync"
	"time"

	"github.com/hashicorp/memberlist"
	"google.golang.org/grpc"
	"gopkg.in/op/go-logging.v1"

	pb "cache/proto/rpc_cache"
	"cache/tools"
)

var log = logging.MustGetLogger("cluster")

// Global singleton. Useful for RPC handlers to be able to access the cluster.
var singleton *Cluster

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
	var err error
	c := memberlist.DefaultLANConfig()
	c.BindPort = port
	c.AdvertisePort = port
	list, err = memberlist.Create(c)
	if err != nil {
		log.Fatalf("Failed to create new memberlist: %s", err)
	}
	n := list.LocalNode()
	log.Notice("Memberlist initialised, this node is %s / %s", n.Name, n.Addr)
	return Cluster{list: list}
}

// JoinCluster joins an existing plz cache cluster.
func (cluster *Cluster) Join(members []string) {
	// Talk to the other nodes to request to join.
	if _, err := cluster.list.Join(members); err != nil {
		log.Fatalf("Failed to join cluster: %s", err)
	}
	for _, node := range cluster.GetMembers() {
		if client, err := cluster.getRPCClient(node.Address); err != nil {
			log.Error("Error getting RPC client for %s: %s", node.Address, err)
		} else if resp, err := client.Join(pb.JoinRequest{
			name:    cluster.list.LocalNode().Name,
			address: cluster.list.LocalNode().Addr,
		}); err != nil {
			log.Error("Error communicating with %s: %s", node.Address, err)
		} else if !resp.Success {
			log.Fatalf("We have not been allowed to join the cluster :(")
		} else {
			cluster.hashStart = resp.HashStart
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
	newNode(list.LocalNode())
	// And there aren't any others yet, so we're done.
}

// GetMembers returns the set of currently known cache members.
func (cluster *Cluster) GetMembers() []*pb.Node {
	// TODO(pebers): this is quadratic so would be bad on large clusters.
	// We might also want to refresh the members in the background if this proves slow?
	for _, m := range list.Members() {
		cluster.newNode(m)
	}
	return cluster.nodes
}

// newNode constructs one of our canonical nodes from a memberlist.Node.
// This includes allocating it hash space.
func (cluster *Cluster) newNode(node *memberlist.Node) *pb.Node {
	for i, n := range cluster.nodes {
		if n.Name == "" || n.Name == node.Name {
			// Available slot. Or, if they identified as an existing node, they can take that space over.
			if n.Name == node.Name {
				log.Warning("Node %s / %s is taking over slot %d", node.Name, node.Addr, i)
			} else {
				log.Notice("Populating node %d: %s / %s", i, node.Name, node.Addr)
			}
			cluster.nodes[i] = &pb.Node{
				Name:      node.Name,
				Address:   node.Addr,
				HashStart: tools.HashPoint(i, len(cluster.nodes)),
				HashEnd:   tools.HashPoint(i+1, len(cluster.nodes)),
			}
			return cluster.nodes[i]
		}
	}
	log.Warning("Node %s / %s attempted to join, but there is no space available.")
	return nil
}

// getRPCClient returns an RPC client for the given server.
func (cluster *Cluster) getRPCClient(address string) (pb.RpcServerClient, error) {
	clientMutex.RLock()
	client, present := clients[address]
	clientMutex.RUnlock()
	if present {
		return client
	}
	// TODO(pebers): add credentials.
	connection, err := grpc.Dial(address, grpc.WithTimeout(5*time.Second), grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	client = pb.NewRpcServerClient(connection)
	clientMutex.Lock()
	clients[address] = client
	clientMutex.Unlock()
	return client, nil
}

// getAddress returns the replica node for the given hash (i.e. whichever one is not us,
// we don't really know for sure when calling this if we are the primary or not).
func (cluster *Cluster) getAlternateNode(hash []byte) string {
	point := tools.Hash(hash)
	if point >= cluster.HashStart && point < cluster.HashEnd {
		// We've got this point, use the alternate.
		point = tools.AlternateHash(hash)
	}
	for _, n := range cluster.nodes {
		if hash >= n.HashStart && hash < n.HashEnd {
			return n.address
		}
	}
	log.Warning("No cluster node found for hash point %d", point)
	return ""
}
