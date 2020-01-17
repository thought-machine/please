// +build !bootstrap

package remote

import (
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/client"
	"google.golang.org/grpc"
)

// This is the real version that we use post-bootstrap.
func (c *Client) dialParams() client.DialParams {
	return client.DialParams{
		Service:            c.state.Config.Remote.URL,
		CASService:         c.state.Config.Remote.CASURL,
		NoSecurity:         !c.state.Config.Remote.Secure,
		TransportCredsOnly: c.state.Config.Remote.Secure,
		DialOpts: []grpc.DialOption{
			grpc.WithStatsHandler(newStatsHandler(c)),
			// Set an arbitrarily large (400MB) max message size so it isn't a limitation.
			grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(419430400)),
		},
	}
}
