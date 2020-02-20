package remote

import (
	"context"
	"encoding/hex"
	"os"

	"github.com/bazelbuild/remote-apis-sdks/go/pkg/chunker"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/digest"
	pb "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"golang.org/x/sync/errgroup"
)

// chunkSize is the size of a chunk that we send when using the ByteStream APIs.
const chunkSize = 128 * 1024

// maxNumBlobs is the maximum number of blobs we request in a batch.
// This is arbitrary but designed to help servers that otherwise get overwhelmed trying to handle
// hundreds at a time.
const maxNumBlobs = 100

// A blob represents something to be uploaded to the remote server.
// It contains the digest of each plus its content or a filename to read it from.
// If the filename is present then the digest's hash may not be populated.
type blob struct {
	Digest *pb.Digest
	Data   []byte
	File   string
	Mode   os.FileMode // Only used when receiving blobs, to determine what the output file mode should be
}

func (b *blob) Chunker(maxChunkSize int) *chunker.Chunker {
	if len(b.Data) != 0 {
		// TODO(peterebden): This will re-digest the blob which seems wasteful.
		return chunker.NewFromBlob(b.Data, maxChunkSize)
	}
	return chunker.NewFromFile(b.File, digest.NewFromProtoUnvalidated(b.Digest), maxChunkSize)
}

// uploadBlobs uploads a series of blobs to the remote.
// It handles all the logic around the various upload methods etc.
// The given function is a callback that receives a channel to send these blobs on; it
// should close it when finished.
func (c *Client) uploadBlobs(f func(ch chan<- *blob) error) error {
	const buffer = 10 // Buffer it a bit but don't get too far ahead.
	ch := make(chan *blob, buffer)
	var g errgroup.Group
	g.Go(func() error { return f(ch) })
	chomks := []*chunker.Chunker{}
	for b := range ch {
		// The actual hash might or might not be set.
		if len(b.Digest.Hash) == 0 {
			h, err := c.state.PathHasher.Hash(b.File, false, true)
			if err != nil {
				return err
			}
			b.Digest.Hash = hex.EncodeToString(h)
		}
		chomks = append(chomks, b.Chunker(int(c.client.ChunkMaxSize)))
	}
	if err := g.Wait(); err != nil {
		return err
	}
	// TODO(peterebden): This timeout is kind of arbitrary since it represents a lot of requests.
	ctx, cancel := context.WithTimeout(context.Background(), 10*c.reqTimeout)
	defer cancel()
	return c.client.UploadIfMissing(ctx, chomks...)
}
