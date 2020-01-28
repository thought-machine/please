package remote

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	pb "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	bs "google.golang.org/genproto/googleapis/bytestream"
	"google.golang.org/grpc/codes"
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

// uploadBlobs uploads a series of blobs to the remote.
// It handles all the logic around the various upload methods etc.
// The given function is a callback that receives a channel to send these blobs on; it
// should close it when finished.
func (c *Client) uploadBlobs(f func(ch chan<- *blob) error) error {
	const buffer = 10 // Buffer it a bit but don't get too far ahead.
	chIn := make(chan *blob, buffer)
	chOut := make(chan *blob, buffer)
	var g errgroup.Group
	g.Go(func() error { return f(chIn) })
	g.Go(func() error { return c.reallyUploadBlobs(chOut) })
	// We use this to deduplicate outgoing blobs; there are cases of repos with hundreds of
	// copies of the same file, and there's no point checking over and over.
	// We don't do it at a more global level but this is *probably* sufficient for now.
	seen := map[string]struct{}{}

	// This function filters a set of blobs through FindMissingBlobs to find out which
	// ones we actually need to upload. The assumption is that most of the time the
	// server will already have a subset of the blobs we're going to upload so we don't
	// want to send them all again.
	filter := func(blobs []*blob) {
		req := &pb.FindMissingBlobsRequest{
			InstanceName: c.instance,
			BlobDigests:  make([]*pb.Digest, 0, len(blobs)),
		}
		m := make(map[string]*blob, len(blobs))
		for _, b := range blobs {
			if _, present := seen[b.Digest.Hash]; !present {
				seen[b.Digest.Hash] = struct{}{}
				req.BlobDigests = append(req.BlobDigests, b.Digest)
				m[b.Digest.Hash] = b
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), c.reqTimeout)
		defer cancel()
		resp, err := c.client.FindMissingBlobs(ctx, req)
		if err != nil {
			log.Warning("Error filtering blobs for remote execution: %s", err)
			// Continue and send all of these, it is not necessarily fatal (although it
			// will probably blow up later, but easier not to handle that error here)
			for _, b := range blobs {
				chOut <- b
			}
			return
		}
		for _, d := range resp.MissingBlobDigests {
			chOut <- m[d.Hash]
		}
	}

	// Buffer them up a bit, the request supports checking multiple at once which is
	// likely more efficient than doing them all one at a time.
	blobs := make([]*blob, 0, buffer)
	for b := range chIn {
		// The actual hash might or might not be set.
		if len(b.Digest.Hash) == 0 {
			h, err := c.state.PathHasher.Hash(b.File, false, true)
			if err != nil {
				return err
			}
			b.Digest.Hash = hex.EncodeToString(h)
		}
		blobs = append(blobs, b)
		if len(blobs) == buffer {
			filter(blobs)
			blobs = blobs[:0]
		}
	}
	if len(blobs) != 0 {
		filter(blobs)
	}
	close(chOut)
	return g.Wait()
}

// reallyUploadBlobs actually does the upload of the individual blobs, after they have
// been filtered through FindMissingBlobs.
func (c *Client) reallyUploadBlobs(ch <-chan *blob) error {
	// Important that if we exit this function with an error we still exhaust the channel,
	// otherwise we can hang other goroutines that are trying to write to it.
	defer exhaustChannel(ch)
	reqs := []*pb.BatchUpdateBlobsRequest_Request{}
	var totalSize int64
	for b := range ch {
		if b.Digest.SizeBytes > c.maxBlobBatchSize {
			// This blob individually exceeds the size, have to use this
			// ByteStream malarkey instead.
			if err := c.storeByteStream(b); err != nil {
				return err
			}
			continue
		} else if b.Digest.SizeBytes+totalSize > c.maxBlobBatchSize {
			// We have exceeded the total but this blob on its own is OK.
			// Send what we have so far then deal with this one.
			if err := c.sendBlobs(reqs); err != nil {
				return err
			}
			reqs = []*pb.BatchUpdateBlobsRequest_Request{}
			totalSize = 0
		}
		// This file is small enough to be read & stored as part of the request.
		// Similarly the data might or might not be available.
		if b.Data == nil {
			data, err := ioutil.ReadFile(b.File)
			if err != nil {
				return err
			}
			b.Data = data
		}
		reqs = append(reqs, &pb.BatchUpdateBlobsRequest_Request{
			Digest: b.Digest,
			Data:   b.Data,
		})
		totalSize += b.Digest.SizeBytes
	}
	if len(reqs) > 0 {
		return c.sendBlobs(reqs)
	}
	return nil
}

// sendBlobs dispatches a set of blobs to the remote CAS server.
func (c *Client) sendBlobs(reqs []*pb.BatchUpdateBlobsRequest_Request) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.reqTimeout)
	defer cancel()
	resp, err := c.client.BatchUpdateBlobs(ctx, &pb.BatchUpdateBlobsRequest{
		InstanceName: c.instance,
		Requests:     reqs,
	})
	if err != nil {
		return err
	}
	// TODO(peterebden): this is not really great handling - we should really use Details
	//                   instead of Message (since this ends up being user-facing) and
	//                   shouldn't just take the first one. This will do for now though.
	for _, r := range resp.Responses {
		if r.Status.Code != int32(codes.OK) {
			return convertError(r.Status)
		}
	}
	return nil
}

// storeByteStream sends a single file as a bytestream. This is required when
// it's over the size limit for BatchUpdateBlobs.
func (c *Client) storeByteStream(b *blob) error {
	// It's probably rare but we might have the contents in memory already at this point.
	// (this shouldn't be a file but could be a serialised proto for example; that's
	// hopefully not common but we do need to handle it here).
	if b.Data != nil {
		return c.reallyStoreByteStream(b, bytes.NewReader(b.Data))
	}
	// Otherwise we need to read the file now.
	f, err := os.Open(b.File)
	if err != nil {
		return err
	}
	defer f.Close()
	return c.reallyStoreByteStream(b, f)
}

func (c *Client) reallyStoreByteStream(b *blob, r io.ReadSeeker) error {
	name := c.byteStreamUploadName(b.Digest)
	ctx, cancel := context.WithTimeout(context.Background(), c.reqTimeout)
	defer cancel()
	stream, err := c.client.Write(ctx)
	if err != nil {
		return err
	}
	offset := 0
	buf := make([]byte, chunkSize)
	for {
		n, err := r.Read(buf)
		if err == io.EOF {
			break
		} else if err != nil {
			// TODO(peterebden): Error handling & retryability (i.e. it's possible to resume
			//                   a failed upload).
			return err
		} else if err := stream.Send(&bs.WriteRequest{
			ResourceName: name,
			WriteOffset:  int64(offset),
			Data:         buf[:n],
		}); err != nil {
			return err
		}
		offset += n
	}
	if err := stream.Send(&bs.WriteRequest{FinishWrite: true, WriteOffset: int64(offset)}); err != nil {
		return err
	}
	_, err = stream.CloseAndRecv()
	return err
}

// byteStreamUploadName returns the resource name for a file uploaded
// as a bytestream. The scheme is specified by the remote execution API.
func (c *Client) byteStreamUploadName(digest *pb.Digest) string {
	u, _ := uuid.NewRandom()
	name := fmt.Sprintf("uploads/%s/blobs/%s/%d", u, digest.Hash, digest.SizeBytes)
	if c.instance != "" {
		name = c.instance + "/" + name
	}
	return name
}
