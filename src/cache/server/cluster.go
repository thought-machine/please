// Contains functions for dealing with a cluster of plz cache nodes.
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

package server

import (
	"sync"

	"github.com/hashicorp/memberlist"
	"google.golang.org/grpc"

	pb "cache/proto/rpc_cache"
	"cache/tools"
)

var list *memberlist.Memberlist

// nodes is a list of nodes that is initialised by the original seed
// and replicated between any other nodes that join after.
var nodes []*pb.ListResponse_Node

// JoinCluster joins an existing plz cache cluster.
func JoinCluster(port int, members []string) {
	if _, err := initCluster(port).Join(members); err != nil {
		log.Fatalf("Failed to join cluster: %s", err)
	}
	// Talk to the other nodes to request to join.
	conn, err := grpc.Dial(config.Cache.RpcUrl, opts...)
}

// InitCluster seeds a new plz cache cluster.
func InitCluster(port, size int) {
	initCluster(port)
	// Create the node list
	nodes = make([]*pb.ListResponse_Node, size)
	// We're node 0
	newNode(list.LocalNode())
}

// GetMembers returns the set of currently known cache members.
func GetMembers() []*pb.ListResponse_Node {
	l := list.Members()
	ret := make([]*pb.ListResponse_Node, len(l))
	for i, m := range l {
		ret[i] = &pb.ListResponse_Node{
			Name:    m.Name,
			Address: m.Addr.String(),
		}
	}
	return ret
}

// IsClustered returns true if the server is currently part of a cluster.
func IsClustered() bool {
	return list != nil
}

func initCluster(port int) *memberlist.Memberlist {
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
	return list
}

// newNode constructs one of our canonical nodes from a memberlist.Node.
// This includes allocating it hash space.
func newNode(node *memberlist.Node) *pb.ListResponse_Node {
	for i, n := range nodes {
		if n.Name == "" || n.Name == node.Name {
			// Available slot. Or, if they identified as an existing node, they can take that space over.
			if n.Name == node.Name {
				log.Warning("Node %s / %s is taking over slot %d", node.Name, node.Addr, i)
			} else {
				log.Notice("Populating node %d: %s / %s", i, node.Name, node.Addr)
			}
			nodes[i] = &pb.ListResponse_Node{
				Name:      node.Name,
				Address:   node.Addr,
				HashStart: tools.HashPoint(i, len(nodes)),
				HashEnd:   tools.HashPoint(i+1, len(nodes)),
			}
			return nodes[i]
		}
	}
	log.Warning("Node %s / %s attempted to join, but there is no space available.")
	return &pb.ListResponse_Node{}
}
