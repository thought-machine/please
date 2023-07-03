package remote

import (
	"context"

	"github.com/bazelbuild/remote-apis-sdks/go/pkg/uploadinfo"
	"golang.org/x/sync/errgroup"
)

// uploadBlobs uploads a series of blobs to the remote.
// It handles all the logic around the various upload methods etc.
// The given function is a callback that receives a channel to send these blobs on; it
// should close it when finished.
func (c *Client) uploadBlobs(ctx context.Context, f func(ch chan<- *uploadinfo.Entry) error) error {
	const buffer = 10 // Buffer it a bit but don't get too far ahead.
	ch := make(chan *uploadinfo.Entry, buffer)
	var g errgroup.Group
	g.Go(func() error { return f(ch) })
	chomks := []*uploadinfo.Entry{}
	for chomk := range ch {
		chomks = append(chomks, chomk)
	}
	if err := g.Wait(); err != nil {
		return err
	}
	return c.uploadIfMissing(ctx, chomks)
}

func (c *Client) uploadIfMissing(ctx context.Context, chomks []*uploadinfo.Entry) error {
	filtered := c.filterEntries(chomks)
	if len(filtered) == 0 {
		return nil
	} else if _, _, err := c.client.UploadIfMissing(ctx, filtered...); err != nil {
		return err
	}
	c.existingBlobMutex.Lock()
	defer c.existingBlobMutex.Unlock()
	for _, entry := range filtered {
		c.existingBlobs[entry.Digest.Hash] = struct{}{}
	}
	return nil
}

func (c *Client) filterEntries(entries []*uploadinfo.Entry) []*uploadinfo.Entry {
	ret := make([]*uploadinfo.Entry, 0, len(entries))
	c.existingBlobMutex.Lock()
	defer c.existingBlobMutex.Unlock()
	for _, entry := range entries {
		if _, present := c.existingBlobs[entry.Digest.Hash]; !present {
			ret = append(ret, entry)
		}
	}
	return ret
}
