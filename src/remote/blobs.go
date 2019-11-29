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
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	bs "google.golang.org/genproto/googleapis/bytestream"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/thought-machine/please/src/fs"
)

// chunkSize is the size of a chunk that we send when using the ByteStream APIs.
const chunkSize = 128 * 1024

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

	// This function filters a set of blobs through FindMissingBlobs to find out which
	// ones we actually need to upload. The assumption is that most of the time the
	// server will already have a subset of the blobs we're going to upload so we don't
	// want to send them all again.
	filter := func(blobs []*blob) {
		req := &pb.FindMissingBlobsRequest{
			InstanceName: c.instance,
			BlobDigests:  make([]*pb.Digest, len(blobs)),
		}
		m := make(map[string]*blob, len(blobs))
		for i, b := range blobs {
			req.BlobDigests[i] = b.Digest
			m[b.Digest.Hash] = b
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

// receiveBlobs retrieves a set of blobs from the remote CAS server.
func (c *Client) receiveBlobs(digests []*pb.Digest, filenames map[string]string, modes map[string]os.FileMode) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.reqTimeout)
	defer cancel()
	resp, err := c.client.BatchReadBlobs(ctx, &pb.BatchReadBlobsRequest{
		InstanceName: c.instance,
		Digests:      digests,
	})
	if err != nil {
		return err
	}
	// TODO(peterebden): as above, could probably handle this a bit better.
	for _, r := range resp.Responses {
		if r.Status.Code != int32(codes.OK) {
			return fmt.Errorf("%s", r.Status.Message)
		}
		filename := filenames[r.Digest.Hash]
		mode := modes[r.Digest.Hash]
		if err := fs.EnsureDir(filename); err != nil {
			return err
		} else if err := fs.WriteFile(bytes.NewReader(r.Data), filename, mode); err != nil {
			return err
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

// byteStreamDownloadName returns the resource name for a file downloaded from
// the API as a bytestream. Of course this is different to the one it's uploaded under.
func (c *Client) byteStreamDownloadName(digest *pb.Digest) string {
	name := fmt.Sprintf("blobs/%s/%d", digest.Hash, digest.SizeBytes)
	if c.instance != "" {
		name = c.instance + "/" + name
	}
	return name
}

// downloadBlobs downloads a series of blobs from the CAS server.
// Each blob given must have the File and Digest properties completely set, but
// Data is not required.
// The given function is a callback that receives a channel to send these blobs on; it
// should close it when finished.
func (c *Client) downloadBlobs(ctx context.Context, f func(ch chan<- *blob) error) error {
	ch := make(chan *blob, 10)
	done := make(chan struct{})
	var g errgroup.Group
	go func() error {
		defer close(done)
		g.Go(func() error { return f(ch) })

		digests := []*pb.Digest{}
		filenames := map[string]string{}  // map of hash -> output filename
		modes := map[string]os.FileMode{} // map of hash -> file mode
		var totalSize int64
		for b := range ch {
			filenames[b.Digest.Hash] = b.File
			modes[b.Digest.Hash] = b.Mode
			if b.Digest.SizeBytes > c.maxBlobBatchSize || !c.canBatchBlobReads {
				// This blob individually exceeds the size, have to use this
				// ByteStream malarkey instead.
				if err := c.retrieveByteStream(b); err != nil {
					return err
				}
				continue
			} else if b.Digest.SizeBytes+totalSize > c.maxBlobBatchSize {
				// We have exceeded the total but this blob on its own is OK.
				// Receive what we have so far then deal with this one.
				if err := c.receiveBlobs(digests, filenames, modes); err != nil {
					return err
				}
				digests = []*pb.Digest{}
				totalSize = 0
			}
			digests = append(digests, b.Digest)
			totalSize += b.Digest.SizeBytes
		}
		// If there are any digests left over, download them now
		if len(digests) > 0 {
			if err := c.receiveBlobs(digests, filenames, modes); err != nil {
				return err
			}
		}
		return nil
	}()

	select {
	case <-done:
		return g.Wait()
	case <-ctx.Done():
		return fmt.Errorf("timed out retrieving artifact from remote cache")
	}

}

// retrieveByteStream receives a file back from the server as a byte stream.
func (c *Client) retrieveByteStream(b *blob) error {
	if b.Digest == nil {
		return fmt.Errorf("can't retrieve byte stream from nil digest")
	}
	r, err := c.readByteStream(b.Digest)
	if err != nil {
		return err
	}
	defer r.Close()
	return fs.WriteFile(r, b.File, b.Mode)
}

// readByteStream returns a reader for a bytestream for the given digest.
func (c *Client) readByteStream(digest *pb.Digest) (io.ReadCloser, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.reqTimeout)
	stream, err := c.client.Read(ctx, &bs.ReadRequest{
		ResourceName: c.byteStreamDownloadName(digest),
	})
	if err != nil {
		cancel()
		return nil, err
	}
	return &byteStreamReader{stream: stream, cancel: cancel, digest: digest}, nil
}

// readAllByteStream returns a bytestream read in its entirety.
func (c *Client) readAllByteStream(digest *pb.Digest) ([]byte, error) {
	r, err := c.readByteStream(digest)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return ioutil.ReadAll(r)
}

// readByteStreamToProto reads an entire bytestream and deserialises it into a message.
func (c *Client) readByteStreamToProto(digest *pb.Digest, msg proto.Message) error {
	b, err := c.readAllByteStream(digest)
	if err != nil {
		return err
	}
	return proto.Unmarshal(b, msg)
}

// checkBatchReadBlobs sends a fake request to verify if BatchReadBlobs is supported
// (it is not on some servers, e.g. buildbarn).
func (c *Client) checkBatchReadBlobs() bool {
	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()
	_, err := c.client.BatchReadBlobs(ctx, &pb.BatchReadBlobsRequest{
		InstanceName: c.instance,
	})
	return status.Code(err) != codes.Unimplemented
}

// A byteStreamReader abstracts over the bytestream gRPC API to turn it into an
// io.Reader which we can then pass to other things which are ignorant of its true nature.
type byteStreamReader struct {
	stream bs.ByteStream_ReadClient
	cancel func()
	buf    []byte
	digest *pb.Digest
}

// Read implements the io.Reader interface
func (r *byteStreamReader) Read(into []byte) (int, error) {
	l := len(into)
	for l > len(r.buf) {
		resp, err := r.stream.Recv()
		if err == io.EOF {
			copy(into, r.buf)
			return len(r.buf), err
		} else if err != nil {
			log.Debug("Error downloading blob for %s/%d: %s", r.digest.Hash, r.digest.SizeBytes, err)
			return 0, err
		}
		r.buf = append(r.buf, resp.Data...)
	}
	copy(into, r.buf[:l])
	r.buf = r.buf[l:]
	return l, nil
}

// Close implements the Closer part of io.ReadCloser
func (r *byteStreamReader) Close() error {
	r.cancel()
	return nil
}
