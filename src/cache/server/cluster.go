// Contains functions for dealing with a cluster of plz cache nodes.
//
// Clustering the cache provides redundancy and increased performance
// for large caches. Right now the functionality is a little limited,
// there's no online rehashing and the replication factor is fixed at 2.
//
// The general approach here errs heavily on the side of simplicity and
// less on zero-downtime reliability since, at the end of the day, this
// is only a cache server.

package server

import (
	"github.com/hashicorp/memberlist"

	pb "cache/proto/rpc_cache"
)

var list *memberlist.Memberlist

// JoinCluster joins an existing plz cache cluster.
func JoinCluster(port int, members []string) {
	if _, err := initCluster(port).Join(members); err != nil {
		log.Fatalf("Failed to join cluster: %s", err)
	}
}

// InitCluster seeds a new plz cache cluster.
func InitCluster(port int) {
	initCluster(port)
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
