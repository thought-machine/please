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
	"strconv"
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

	// size is the expected number of nodes in the cluster.
	size int

	// hashStart and hashEnd are the endpoints in the hash space for this client.
	hashStart uint32
	hashEnd   uint32
}

// NewCluster creates a new Cluster object and starts listening on the given port.
func NewCluster(port, rpcPort int, name string) *Cluster {
	c := memberlist.DefaultLANConfig()
	c.BindPort = port
	c.AdvertisePort = port
	c.Delegate = &delegate{port: rpcPort}
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
	for _, node := range cluster.list.Members() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if client, err := cluster.getRPCClient(node.Addr.String() + ":" + string(node.Meta)); err != nil {
			log.Error("Error getting RPC client for %s: %s", node.Addr, err)
		} else if resp, err := client.Join(ctx, &pb.JoinRequest{
			Name:    cluster.list.LocalNode().Name,
			Address: cluster.list.LocalNode().Addr.String(),
		}); err != nil {
			log.Error("Error communicating with %s: %s", node.Addr, err)
		} else if !resp.Success {
			log.Fatalf("We have not been allowed to join the cluster :(")
		} else {
			cluster.hashStart = resp.HashBegin
			cluster.hashEnd = resp.HashEnd
			cluster.nodes = resp.Nodes
			cluster.size = int(resp.Size)
			return
		}
	}
	log.Fatalf("Unable to contact any other cluster members")
}

// InitCluster seeds a new plz cache cluster.
func (cluster *Cluster) Init(size int) {
	cluster.size = size
	// We're node 0
	node := cluster.newNode(cluster.list.LocalNode())
	cluster.hashStart = node.HashBegin
	cluster.hashEnd = node.HashEnd
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
	newNode := func(i int) *pb.Node {
		return &pb.Node{
			Name:      node.Name,
			Address:   node.Addr.String() + ":" + string(node.Meta),
			HashBegin: tools.HashPoint(i, cluster.size),
			HashEnd:   tools.HashPoint(i+1, cluster.size),
		}
	}
	for i, n := range cluster.nodes {
		if n.Name == "" || n.Name == node.Name {
			// Available slot. Or, if they identified as an existing node, they can take that space over.
			if n.Name == node.Name {
				log.Warning("Node %s / %s is taking over slot %d", node.Name, node.Addr, i)
			} else {
				log.Notice("Populating node %d: %s / %s", i, node.Name, node.Addr)
			}
			cluster.nodes[i] = newNode(i)
			return cluster.nodes[i]
		}
	}
	if len(cluster.nodes) < cluster.size {
		node := newNode(len(cluster.nodes))
		cluster.nodes = append(cluster.nodes, node)
		return node
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

// getAlternateNode returns the replica node for the given hash (i.e. whichever one is not us,
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
	log.Info("Replicating artifact to node %s", address)
	cluster.replicate(address, req.Os, req.Arch, req.Hash, false, req.Artifacts)
}

// DeleteArtifacts deletes artifacts from all other nodes.
func (cluster *Cluster) DeleteArtifacts(req *pb.DeleteRequest) {
	for _, node := range cluster.GetMembers() {
		// Don't forward request to ourselves...
		if node.HashBegin != cluster.hashStart || node.HashEnd != cluster.hashEnd {
			log.Info("Forwarding delete request to node %s", node.Address)
			cluster.replicate(node.Address, req.Os, req.Arch, nil, true, req.Artifacts)
		}
	}
}

func (cluster *Cluster) replicate(address, os, arch string, hash []byte, delete bool, artifacts []*pb.Artifact) {
	client, err := cluster.getRPCClient(address)
	if err != nil {
		log.Error("Failed to get RPC client for %s: %s", address, err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if resp, err := client.Replicate(ctx, &pb.ReplicateRequest{
		Artifacts: artifacts,
		Os:        os,
		Arch:      arch,
		Hash:      hash,
		Delete:    delete,
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
		log.Warning("Rejected join request from %s", req.Address)
		return &pb.JoinResponse{Success: false}
	}
	return &pb.JoinResponse{
		Success:   true,
		HashBegin: node.HashBegin,
		HashEnd:   node.HashEnd,
		Nodes:     cluster.GetMembers(),
		Size:      int32(cluster.size),
	}
}

// A delegate is our implementation of memberlist's Delegate interface.
// Somewhat awkwardly we have to implement the whole thing to provide metadata for our node,
// which we only really need to do to communicate our RPC port.
type delegate struct {
	port int
}

func (d *delegate) NodeMeta(limit int) []byte {
	return []byte(strconv.Itoa(d.port))
}

func (d *delegate) NotifyMsg([]byte)                           {}
func (d *delegate) GetBroadcasts(overhead, limit int) [][]byte { return nil }
func (d *delegate) LocalState(join bool) []byte                { return nil }
func (d *delegate) MergeRemoteState(buf []byte, join bool)     {}
