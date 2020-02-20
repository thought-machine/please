package remote

import (
	"context"

	"github.com/bazelbuild/remote-apis-sdks/go/pkg/chunker"
	"golang.org/x/sync/errgroup"
)

// uploadBlobs uploads a series of blobs to the remote.
// It handles all the logic around the various upload methods etc.
// The given function is a callback that receives a channel to send these blobs on; it
// should close it when finished.
func (c *Client) uploadBlobs(f func(ch chan<- *chunker.Chunker) error) error {
	const buffer = 10 // Buffer it a bit but don't get too far ahead.
	ch := make(chan *chunker.Chunker, buffer)
	var g errgroup.Group
	g.Go(func() error { return f(ch) })
	chomks := []*chunker.Chunker{}
	for chomk := range ch {
		chomks = append(chomks, chomk)
	}
	if err := g.Wait(); err != nil {
		return err
	}
	// TODO(peterebden): This timeout is kind of arbitrary since it represents a lot of requests.
	ctx, cancel := context.WithTimeout(context.Background(), 10*c.reqTimeout)
	defer cancel()
	return c.client.UploadIfMissing(ctx, chomks...)
}
