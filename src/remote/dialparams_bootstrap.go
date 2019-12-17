// +build bootstrap

package remote

import "github.com/bazelbuild/remote-apis-sdks/go/pkg/client"

// This is a special-cased version during bootstrap where we have to use the vanilla upstream
// library which doesn't let us specify dial options.
// TODO(peterebden): This should go away if we upstream this change.
func (c *Client) dialParams() client.DialParams {
	return client.DialParams{
		Service:            c.state.Config.Remote.URL,
		CASService:         c.state.Config.Remote.CASURL,
		NoSecurity:         !c.state.Config.Remote.Secure,
		TransportCredsOnly: c.state.Config.Remote.Secure,
	}
}
