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
	return c.uploadIfMissing(context.Background(), chomks...)
}

func (c *Client) uploadIfMissing(ctx context.Context, chomks ...*chunker.Chunker) error {
	filtered := c.filterChunks(chomks)
	if len(filtered) == 0 {
		return nil
	} else if err := c.client.UploadIfMissing(ctx, filtered...); err != nil {
		return err
	}
	c.existingBlobMutex.Lock()
	defer c.existingBlobMutex.Unlock()
	for _, chunk := range chomks {
		c.existingBlobs[chunk.Digest().Hash] = struct{}{}
	}
	return nil
}

func (c *Client) filterChunks(chomks []*chunker.Chunker) []*chunker.Chunker {
	ret := make([]*chunker.Chunker, 0, len(chomks))
	c.existingBlobMutex.Lock()
	defer c.existingBlobMutex.Unlock()
	for _, chunk := range chomks {
		if _, present := c.existingBlobs[chunk.Digest().Hash]; !present {
			ret = append(ret, chunk)
		}
	}
	return ret
}
