// Package remote provides our interface to the Google remote execution APIs
// (https://github.com/bazelbuild/remote-apis) which Please can use to distribute
// work to remote servers.
package remote

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	pb "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/bazelbuild/remote-apis/build/bazel/semver"
	"github.com/golang/protobuf/ptypes"
	"github.com/grpc-ecosystem/go-grpc-middleware/retry"
	bs "google.golang.org/genproto/googleapis/bytestream"
	"google.golang.org/genproto/googleapis/longrunning"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip" // Registers the gzip compressor at init
	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

var log = logging.MustGetLogger("remote")

// Timeout to initially contact the server.
const dialTimeout = 5 * time.Second

// Timeout for actual requests
const reqTimeout = 2 * time.Minute

// Maximum number of times we retry a request.
const maxRetries = 3

// The API version we support.
var apiVersion = semver.SemVer{Major: 2}

// A Client is the interface to the remote API.
//
// It provides a higher-level interface over the specific RPCs available.
type Client struct {
	actionCacheClient pb.ActionCacheClient
	storageClient     pb.ContentAddressableStorageClient
	bsClient          bs.ByteStreamClient
	execClient        pb.ExecutionClient
	initOnce          sync.Once
	state             *core.BuildState
	err               error // for initialisation
	instance          string

	// Server-sent cache properties
	maxBlobBatchSize  int64
	cacheWritable     bool
	canBatchBlobReads bool // This isn't supported by all servers.

	// Cache this for later
	bashPath string
}

// instance is the singleton client instance for Get()
var instance *Client

// New returns a new Client instance.
// It begins the process of contacting the remote server but does not wait for it.
func New(state *core.BuildState) *Client {
	c := &Client{state: state, instance: state.Config.Remote.Instance}
	go c.CheckInitialised() // Kick off init now, but we don't have to wait for it.
	return c
}

// Get is like New but it populates and retrieves a singleton instance.
func Get(state *core.BuildState) *Client {
	if instance == nil {
		instance = New(state)
	}
	return instance
}

// CheckInitialised checks that the client has connected to the server correctly.
func (c *Client) CheckInitialised() error {
	c.initOnce.Do(c.init)
	return c.err
}

// init is passed to the sync.Once to do the actual initialisation.
func (c *Client) init() {
	c.err = func() error {
		// TODO(peterebden): We may need to add the ability to have multiple URLs which we
		//                   would then query for capabilities to discover which is which.
		// TODO(peterebden): Add support for TLS.
		conn, err := grpc.Dial(c.state.Config.Remote.URL.String(),
			grpc.WithTimeout(dialTimeout),
			grpc.WithInsecure(),
			grpc.WithUnaryInterceptor(grpc_retry.UnaryClientInterceptor(grpc_retry.WithMax(maxRetries))))
		if err != nil {
			return err
		}
		// Query the server for its capabilities. This tells us whether it is capable of
		// execution, caching or both.
		ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
		defer cancel()
		resp, err := pb.NewCapabilitiesClient(conn).GetCapabilities(ctx, &pb.GetCapabilitiesRequest{
			InstanceName: c.instance,
		})
		if err != nil {
			return err
		}
		if lessThan(&apiVersion, resp.LowApiVersion) || lessThan(resp.HighApiVersion, &apiVersion) {
			return fmt.Errorf("Unsupported API version; we require %s but server only supports %s - %s", printVer(&apiVersion), printVer(resp.LowApiVersion), printVer(resp.HighApiVersion))
		}
		caps := resp.CacheCapabilities
		if caps == nil {
			return fmt.Errorf("Cache capabilities not supported by server (we do not support execution-only servers)")
		}
		if err := c.chooseDigest(caps.DigestFunction); err != nil {
			return err
		}
		if caps.ActionCacheUpdateCapabilities != nil {
			c.cacheWritable = caps.ActionCacheUpdateCapabilities.UpdateEnabled
		}
		c.maxBlobBatchSize = caps.MaxBatchTotalSizeBytes
		if c.maxBlobBatchSize == 0 {
			// No limit was set by the server, assume we are implicitly limited to 4MB (that's
			// gRPC's limit which most implementations do not seem to override). Round it down a
			// bit to allow a bit of serialisation overhead etc.
			c.maxBlobBatchSize = 4000000
		}
		c.actionCacheClient = pb.NewActionCacheClient(conn)
		c.storageClient = pb.NewContentAddressableStorageClient(conn)
		c.bsClient = bs.NewByteStreamClient(conn)
		// Look this up just once now.
		bash, err := core.LookBuildPath("bash", c.state.Config)
		c.bashPath = bash
		c.canBatchBlobReads = c.checkBatchReadBlobs()
		log.Debug("Remote execution client initialised for storage")
		// Now check if it can do remote execution
		if caps := resp.ExecutionCapabilities; caps != nil && c.state.Config.Remote.NumExecutors > 0 {
			if err := c.chooseDigest([]pb.DigestFunction_Value{caps.DigestFunction}); err != nil {
				return err
			} else if !caps.ExecEnabled {
				return fmt.Errorf("Remote execution not enabled for this server")
			}
			c.execClient = pb.NewExecutionClient(conn)
			log.Debug("Remote execution client initialised for execution")
		}
		return err
	}()
	if c.err != nil {
		log.Error("Error setting up remote execution client: %s", c.err)
	}
}

// chooseDigest selects a digest function that we will use.w
func (c *Client) chooseDigest(fns []pb.DigestFunction_Value) error {
	for _, fn := range fns {
		// Right now the only choice we can make generally is SHA1.
		// In future we might let ourselves be guided by this and choose something else
		// that matches the server (but that implies that all targets have to be hashed
		// with it, hence we'd have to synchronously initialise against the server, and
		// it's unclear whether this will be an issue in practice anyway).
		if fn == pb.DigestFunction_SHA1 {
			return nil
		}
	}
	return fmt.Errorf("No acceptable hash function available; server supports %s but we require SHA1", fns)
}

// Store stores a set of artifacts for a single build target.
func (c *Client) Store(target *core.BuildTarget, key []byte, metadata *core.BuildMetadata, files []string) error {
	if err := c.CheckInitialised(); err != nil {
		return err
	}
	ar := &pb.ActionResult{
		// We never cache any failed actions so ExitCode is implicitly 0.
		ExecutionMetadata: &pb.ExecutedActionMetadata{
			Worker:                      c.state.Config.Remote.Name,
			OutputUploadStartTimestamp:  toTimestamp(time.Now()),
			ExecutionStartTimestamp:     toTimestamp(metadata.StartTime),
			ExecutionCompletedTimestamp: toTimestamp(metadata.EndTime),
		},
	}
	outDir := target.OutDir()
	if err := c.uploadBlobs(func(ch chan<- *blob) error {
		defer close(ch)
		for _, file := range files {
			file = path.Join(outDir, file)
			info, err := os.Lstat(file)
			if err != nil {
				return err
			} else if mode := info.Mode(); mode&os.ModeDir != 0 {
				// It's a directory, needs special treatment
				root, children, err := c.digestDir(file, nil)
				if err != nil {
					return err
				}
				digest := digestMessage(&pb.Tree{
					Root:     root,
					Children: children,
				})
				ar.OutputDirectories = append(ar.OutputDirectories, &pb.OutputDirectory{
					Path:       file,
					TreeDigest: digest,
				})
				continue
			} else if mode&os.ModeSymlink != 0 {
				target, err := os.Readlink(file)
				if err != nil {
					return err
				}
				// TODO(peterebden): Work out if we need to give a shit about
				//                   OutputDirectorySymlinks or not. Seems like we shouldn't
				//                   need to care since symlinks don't know the type of thing
				//                   they point to?
				ar.OutputFileSymlinks = append(ar.OutputFileSymlinks, &pb.OutputSymlink{
					Path:   file,
					Target: target,
				})
				continue
			}
			// It's a real file, bung it onto the channel.
			h, err := c.state.PathHasher.Hash(file, false, true)
			if err != nil {
				return err
			}
			digest := &pb.Digest{
				SizeBytes: info.Size(),
				Hash:      hex.EncodeToString(h),
			}
			ch <- &blob{
				File:   file,
				Digest: digest,
			}
			ar.OutputFiles = append(ar.OutputFiles, &pb.OutputFile{
				Path:   file,
				Digest: digest,
			})
			if len(metadata.Stdout) > 0 {
				h := sha1.Sum(metadata.Stdout)
				digest := &pb.Digest{
					SizeBytes: int64(len(metadata.Stdout)),
					Hash:      hex.EncodeToString(h[:]),
				}
				ch <- &blob{
					Data:   metadata.Stdout,
					Digest: digest,
				}
				ar.StdoutDigest = digest
			}
		}
		return nil
	}); err != nil {
		return err
	}
	// OK, now the blobs are uploaded, we also need to upload the Action itself.
	digest, err := c.uploadAction(target, key, metadata.Test)
	if err != nil {
		return err
	}
	// Now we can use that to upload the result itself.
	ctx, cancel := context.WithTimeout(context.Background(), reqTimeout)
	defer cancel()
	_, err = c.actionCacheClient.UpdateActionResult(ctx, &pb.UpdateActionResultRequest{
		InstanceName: c.instance,
		ActionDigest: digest,
		ActionResult: ar,
	})
	return err
}

// Retrieve fetches back a set of artifacts for a single build target.
// Its outputs are written out to their final locations.
func (c *Client) Retrieve(target *core.BuildTarget, key []byte) (*core.BuildMetadata, error) {
	if err := c.CheckInitialised(); err != nil {
		return nil, err
	}
	isTest := target.State() > core.Built
	needStdout := target.PostBuildFunction != nil && !isTest // We only care in this case.
	inputRoot, err := c.buildInputRoot(target, false, isTest)
	if err != nil {
		return nil, err
	}
	digest := digestMessage(&pb.Action{
		CommandDigest:   digestMessage(c.buildCommand(target, key, isTest)),
		InputRootDigest: digestMessage(inputRoot),
		Timeout:         ptypes.DurationProto(target.BuildTimeout),
	})
	ctx, cancel := context.WithTimeout(context.Background(), reqTimeout)
	defer cancel()
	resp, err := c.actionCacheClient.GetActionResult(ctx, &pb.GetActionResultRequest{
		InstanceName: c.instance,
		ActionDigest: digest,
		InlineStdout: needStdout,
	})
	if err != nil {
		return nil, err
	}
	mode := target.OutMode()
	if err := c.downloadBlobs(func(ch chan<- *blob) error {
		for _, file := range resp.OutputFiles {
			addPerms := extraPerms(file)
			if file.Contents != nil {
				// Inlining must have been requested. Can write it directly.
				if err := fs.EnsureDir(file.Path); err != nil {
					return err
				}
				return fs.WriteFile(bytes.NewReader(file.Contents), file.Path, mode|addPerms)
			}
			ch <- &blob{Digest: file.Digest, File: file.Path, Mode: mode | addPerms}
		}
		close(ch)
		return nil
	}); err != nil {
		return nil, err
	}
	return c.buildMetadata(resp, needStdout, false)
}

// Build executes a remote build of the given target.
func (c *Client) Build(tid int, target *core.BuildTarget, stamp []byte) (*core.BuildMetadata, error) {
	if err := c.CheckInitialised(); err != nil {
		return nil, err
	}
	digest, err := c.uploadAction(target, stamp, false)
	if err != nil {
		return nil, err
	}
	return c.execute(tid, target, digest, target.BuildTimeout, target.PostBuildFunction != nil)
}

// Test executes a remote test of the given target.
// TODO(peterebden): Return test results and coverage info too.
func (c *Client) Test(tid int, state *core.BuildState, target *core.BuildTarget) (*core.BuildMetadata, error) {
	if err := c.CheckInitialised(); err != nil {
		return nil, err
	}
	digest, err := c.uploadAction(target, nil, true)
	if err != nil {
		return nil, err
	}
	return c.execute(tid, target, digest, target.TestTimeout, false)
}

// execute submits an action to the remote executor and monitors its progress.
func (c *Client) execute(tid int, target *core.BuildTarget, digest *pb.Digest, timeout time.Duration, needStdout bool) (*core.BuildMetadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	stream, err := c.execClient.Execute(ctx, &pb.ExecuteRequest{
		InstanceName: c.instance,
		ActionDigest: digest,
	})
	if err != nil {
		return nil, err
	}
	for {
		resp, err := stream.Recv()
		if err != nil {
			// We shouldn't get an EOF here because there's in-channel signalling via Done.
			// TODO(peterebden): On other errors we should be able to reconnect and use
			//                   WaitExecution to rejoin the stream.
			return nil, err
		}
		metadata := &pb.ExecuteOperationMetadata{}
		if err := ptypes.UnmarshalAny(resp.Metadata, metadata); err != nil {
			log.Warning("Failed to deserialise execution metadata: %s", err)
		} else {
			c.updateProgress(tid, target, metadata)
			// TODO(peterebden): At this point we could stream stdout / stderr if the
			//                   user has set --show_all_output.
		}
		if resp.Done {
			switch result := resp.Result.(type) {
			case *longrunning.Operation_Error:
				// We shouldn't really get here - the rex API requires servers to always
				// use the response field instead of error.
				return nil, convertError(result.Error)
			case *longrunning.Operation_Response:
				response := &pb.ExecuteResponse{}
				if err := ptypes.UnmarshalAny(result.Response, response); err != nil {
					log.Error("Failed to deserialise execution response: %s", err)
					return nil, err
				}
				if response.CachedResult {
					c.state.LogBuildResult(tid, target.Label, core.TargetCached, "Cached")
				}
				for k, v := range response.ServerLogs {
					log.Debug("Server log available: %s: %s", k, v.Digest.Hash)
				}
				respErr := convertError(response.Status)
				if resp.Result == nil { // This is optional on failure.
					return nil, respErr
				}
				metadata, err := c.buildMetadata(response.Result, needStdout || respErr != nil, respErr != nil)
				// The original error is higher priority than us trying to retrieve the
				// output of the thing that failed.
				if respErr != nil {
					return metadata, respErr
				}
				return metadata, err
			}
		}
	}
}

// updateProgress updates the progress of a target based on its metadata.
func (c *Client) updateProgress(tid int, target *core.BuildTarget, metadata *pb.ExecuteOperationMetadata) {
	if target.State() > core.Built {
		switch metadata.Stage {
		case pb.ExecutionStage_CACHE_CHECK:
			c.state.LogBuildResult(tid, target.Label, core.TargetBuilding, "Checking cache")
		case pb.ExecutionStage_QUEUED:
			c.state.LogBuildResult(tid, target.Label, core.TargetBuilding, "Queued")
		case pb.ExecutionStage_EXECUTING:
			c.state.LogBuildResult(tid, target.Label, core.TargetBuilding, "Building")
		case pb.ExecutionStage_COMPLETED:
			c.state.LogBuildResult(tid, target.Label, core.TargetBuilt, "Built")
		}
	} else {
		switch metadata.Stage {
		case pb.ExecutionStage_CACHE_CHECK:
			c.state.LogBuildResult(tid, target.Label, core.TargetTesting, "Checking cache")
		case pb.ExecutionStage_QUEUED:
			c.state.LogBuildResult(tid, target.Label, core.TargetTesting, "Queued")
		case pb.ExecutionStage_EXECUTING:
			c.state.LogBuildResult(tid, target.Label, core.TargetTesting, "Testing")
		case pb.ExecutionStage_COMPLETED:
			c.state.LogBuildResult(tid, target.Label, core.TargetTested, "Tested")
		}
	}
}
