// Package cluster contains functions for dealing with a cluster of plz cache nodes.
//
// Clustering the cache provides redundancy and increased performance
// for large caches. Right now the functionality is a little limited,
// there's no online rehashing so the size must be declared and fixed
// up front, and the replication factor is fixed at 2. There's an
// assumption that while nodes might restart, they return with the same
// name which we use to re-identify them.
//
// The general approach here errs heavily on the side of simplicity and
// less on zero-downtime reliability since, at the end of the day, this
// is only a cache server.
package cluster

import (
	"bytes"
	"context"
	"fmt"
	stdlog "log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"github.com/hashicorp/memberlist"
	"google.golang.org/grpc"
	"gopkg.in/op/go-logging.v1"

	pb "github.com/thought-machine/please/src/cache/proto/rpc_cache"
	"github.com/thought-machine/please/src/cache/tools"
)

var log = logging.MustGetLogger("cluster")

// A Cluster handles communication between a set of clustered cache servers.
type Cluster struct {
	list *memberlist.Memberlist
	// nodes is a list of nodes that is initialised by the original seed
	// and replicated between any other nodes that join after.
	nodes []*pb.Node
	// nodeMutex protects access to nodes
	nodeMutex sync.RWMutex

	// clients is a pool of gRPC clients to the other cluster nodes.
	clients map[string]pb.RpcServerClient
	// clientMutex protects concurrent access to clients.
	clientMutex sync.RWMutex

	// size is the expected number of nodes in the cluster.
	size int

	// node is the node corresponding to this instance.
	node *pb.Node

	// hostname is our hostname.
	hostname string
	// name is the name of this cluster node.
	name string
}

// NewCluster creates a new Cluster object and starts listening on the given port.
func NewCluster(port, rpcPort int, name, advertiseAddr string) *Cluster {
	c := memberlist.DefaultLANConfig()
	c.BindPort = port
	c.AdvertisePort = port
	c.Delegate = &delegate{name: name, port: rpcPort}
	c.Logger = stdlog.New(&logWriter{}, "", 0)
	c.AdvertiseAddr = advertiseAddr
	if name != "" {
		c.Name = name
	}
	list, err := memberlist.Create(c)
	if err != nil {
		log.Fatalf("Failed to create new memberlist: %s", err)
	}
	clu := &Cluster{
		clients: map[string]pb.RpcServerClient{},
		name:    name,
		list:    list,
	}
	if hostname, err := os.Hostname(); err == nil {
		clu.hostname = hostname
	}
	n := list.LocalNode()
	log.Notice("Memberlist initialised, this node is %s / %s:%d", n.Name, n.Addr, port)
	return clu
}

// Join joins an existing plz cache cluster.
func (cluster *Cluster) Join(members []string) {
	// Talk to the other nodes to request to join.
	if _, err := cluster.list.Join(members); err != nil {
		log.Fatalf("Failed to join cluster: %s", err)
	}
	for _, node := range cluster.list.Members() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		name, port := cluster.metadata(node)
		if name == cluster.name {
			continue // Don't attempt to join ourselves, we're in the memberlist but can't welcome a new member.
		}
		log.Notice("Attempting to join with %s: %s / %s", name, node.Addr, port)
		if client, err := cluster.getRPCClient(name, node.Addr.String()+port); err != nil {
			log.Error("Error getting RPC client for %s: %s", node.Addr, err)
		} else if resp, err := client.Join(ctx, &pb.JoinRequest{
			Name:    cluster.list.LocalNode().Name,
			Address: cluster.list.LocalNode().Addr.String(),
		}); err != nil {
			log.Error("Error communicating with %s: %s", node.Addr, err)
		} else if !resp.Success {
			log.Fatalf("We have not been allowed to join the cluster :(")
		} else {
			cluster.nodes = resp.Nodes
			cluster.node = resp.Node
			cluster.size = int(resp.Size)
			return
		}
	}
	log.Fatalf("Unable to contact any other cluster members")
}

// metadata breaks metadata from a node into its name and port (with a leading colon).
func (cluster *Cluster) metadata(node *memberlist.Node) (string, string) {
	meta := string(node.Meta)
	idx := strings.IndexRune(meta, ':')
	if idx == -1 {
		return "", ""
	}
	return meta[:idx], meta[idx:]
}

// Init seeds a new plz cache cluster.
func (cluster *Cluster) Init(size int) {
	cluster.size = size
	// We're node 0. The following will add us to the node list.
	cluster.node = cluster.newNode(cluster.list.LocalNode())
	// And there aren't any others yet, so we're done.
}

// GetMembers returns the set of currently known cache members.
func (cluster *Cluster) GetMembers() []*pb.Node {
	// TODO(pebers): this is quadratic so would be bad on large clusters.
	// We might also want to refresh the members in the background if this proves slow?
	for _, m := range cluster.list.Members() {
		cluster.newNode(m)
	}
	return cluster.nodes[:]
}

// newNode constructs one of our canonical nodes from a memberlist.Node.
// This includes allocating it hash space.
func (cluster *Cluster) newNode(node *memberlist.Node) *pb.Node {
	newNode := func(i int) *pb.Node {
		_, port := cluster.metadata(node)
		return &pb.Node{
			Name:      node.Name,
			Address:   node.Addr.String() + port,
			HashBegin: tools.HashPoint(i, cluster.size),
			HashEnd:   tools.HashPoint(i+1, cluster.size),
		}
	}
	cluster.nodeMutex.Lock()
	defer cluster.nodeMutex.Unlock()
	for i, n := range cluster.nodes {
		if n.Name == "" || n.Name == node.Name {
			// Available slot. Or, if they identified as an existing node, they can take that space over.
			if n.Name == node.Name {
				log.Notice("Node %s / %s matched to slot %d", node.Name, node.Addr, i)
			} else {
				log.Notice("Populating node %d: %s / %s", i, node.Name, node.Addr)
			}
			cluster.nodes[i] = newNode(i)
			// Remove any client that might exist for this node so we force a reconnection.
			cluster.clientMutex.Lock()
			defer cluster.clientMutex.Unlock()
			delete(cluster.clients, n.Name)
			return cluster.nodes[i]
		}
	}
	if len(cluster.nodes) < cluster.size {
		node := newNode(len(cluster.nodes))
		cluster.nodes = append(cluster.nodes, node)
		return node
	}
	log.Warning("Node %s / %s attempted to join, but there is no space available [%d / %d].", node.Name, node.Addr, len(cluster.nodes), cluster.size)
	return nil
}

// getRPCClient returns an RPC client for the given server.
func (cluster *Cluster) getRPCClient(name, address string) (pb.RpcServerClient, error) {
	cluster.clientMutex.RLock()
	client, present := cluster.clients[name]
	cluster.clientMutex.RUnlock()
	if present {
		return client, nil
	}
	// TODO(pebers): add credentials.
	connection, err := grpc.Dial(address, grpc.WithTimeout(5*time.Second), grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(grpc_retry.UnaryClientInterceptor(grpc_retry.WithMax(3))))
	if err != nil {
		return nil, err
	}
	client = pb.NewRpcServerClient(connection)
	if name != "" {
		cluster.clientMutex.Lock()
		cluster.clients[name] = client
		cluster.clientMutex.Unlock()
	}
	return client, nil
}

// getAlternateNode returns the replica node for the given hash (i.e. whichever one is not us,
// we don't really know for sure when calling this if we are the primary or not).
func (cluster *Cluster) getAlternateNode(hash []byte) (string, string) {
	point := tools.Hash(hash)
	if point >= cluster.node.HashBegin && point < cluster.node.HashEnd {
		// We've got this point, use the alternate.
		point = tools.AlternateHash(hash)
	}
	cluster.nodeMutex.RLock()
	defer cluster.nodeMutex.RUnlock()
	for _, n := range cluster.nodes {
		if point >= n.HashBegin && point < n.HashEnd {
			return n.Name, n.Address
		}
	}
	log.Warning("No cluster node found for hash point %d", point)
	return "", ""
}

// ReplicateArtifacts replicates artifacts from this node to another.
func (cluster *Cluster) ReplicateArtifacts(req *pb.StoreRequest) {
	name, address := cluster.getAlternateNode(req.Hash)
	if address == "" {
		log.Warning("Couldn't get alternate address, will not replicate artifact")
		return
	}
	log.Info("Replicating artifact to node %s", address)
	cluster.replicate(name, address, req.Os, req.Arch, req.Hash, false, req.Artifacts, req.Hostname)
}

// DeleteArtifacts deletes artifacts from all other nodes.
func (cluster *Cluster) DeleteArtifacts(req *pb.DeleteRequest) {
	for _, node := range cluster.GetMembers() {
		// Don't forward request to ourselves...
		if cluster.node.Name != node.Name {
			log.Info("Forwarding delete request to node %s", node.Address)
			cluster.replicate(node.Name, node.Address, req.Os, req.Arch, nil, true, req.Artifacts, "")
		}
	}
}

func (cluster *Cluster) replicate(name, address, os, arch string, hash []byte, delete bool, artifacts []*pb.Artifact, hostname string) {
	client, err := cluster.getRPCClient(name, address)
	if err != nil {
		log.Error("Failed to get RPC client for %s %s: %s", name, address, err)
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
		Hostname:  hostname,
		Peer:      cluster.hostname,
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
		Success: true,
		Nodes:   cluster.GetMembers(),
		Node:    node,
		Size:    int32(cluster.size),
	}
}

// A delegate is our implementation of memberlist's Delegate interface.
// Somewhat awkwardly we have to implement the whole thing to provide metadata for our node,
// which we only really need to do to communicate our name and RPC port.
type delegate struct {
	name string
	port int
}

func (d *delegate) NodeMeta(limit int) []byte {
	return []byte(d.name + ":" + strconv.Itoa(d.port))
}

func (d *delegate) NotifyMsg([]byte)                           {}
func (d *delegate) GetBroadcasts(overhead, limit int) [][]byte { return nil }
func (d *delegate) LocalState(join bool) []byte                { return nil }
func (d *delegate) MergeRemoteState(buf []byte, join bool)     {}

// A logWriter is a wrapper around our logger to decode memberlist's prefixes into our logging levels.
type logWriter struct{}

// logLevels maps memberlist's prefixes to our logging levels.
var logLevels = map[string]func(format string, args ...interface{}){
	"[ERR]":   log.Errorf,
	"[ERROR]": log.Errorf,
	"[WARN]":  log.Warning,
	"[INFO]":  log.Info,
	"[DEBUG]": log.Debug,
}

// Write implements the io.Writer interface
func (w *logWriter) Write(b []byte) (int, error) {
	for prefix, f := range logLevels {
		if bytes.HasPrefix(b, []byte(prefix)) {
			f(string(bytes.TrimSpace(bytes.TrimPrefix(b, []byte(prefix)))))
			return len(b), nil
		}
	}
	return 0, fmt.Errorf("Couldn't decide how to log %s", string(b))
}
