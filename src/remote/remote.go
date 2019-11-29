// Package remote provides our interface to the Google remote execution APIs
// (https://github.com/bazelbuild/remote-apis) which Please can use to distribute
// work to remote servers.
package remote

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"github.com/bazelbuild/remote-apis-sdks/go/pkg/client"
	pb "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/bazelbuild/remote-apis/build/bazel/semver"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/genproto/googleapis/longrunning"
	_ "google.golang.org/grpc/encoding/gzip" // Registers the gzip compressor at init
	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

var log = logging.MustGetLogger("remote")

// Timeout to initially contact the server.
const dialTimeout = 5 * time.Second

// Maximum number of times we retry a request.
const maxRetries = 3

// The API version we support.
var apiVersion = semver.SemVer{Major: 2}

// A Client is the interface to the remote API.
//
// It provides a higher-level interface over the specific RPCs available.
type Client struct {
	client     *client.Client
	initOnce   sync.Once
	state      *core.BuildState
	reqTimeout time.Duration
	err        error // for initialisation
	instance   string

	// Stored output directories from previously executed targets.
	// This isn't just a cache - it is needed for cases where we don't actually
	// have the files physically on disk.
	outputs     map[core.BuildLabel]*pb.Directory
	outputMutex sync.RWMutex

	// Server-sent cache properties
	maxBlobBatchSize  int64
	cacheWritable     bool
	canBatchBlobReads bool // This isn't supported by all servers.

	// True if we are doing proper remote execution (false if we are caching only)
	remoteExecution bool
	// Platform properties that we will request from the remote.
	// TODO(peterebden): this will need some modification for cross-compiling support.
	platform *pb.Platform

	// Cache this for later
	bashPath string
}

// New returns a new Client instance.
// It begins the process of contacting the remote server but does not wait for it.
func New(state *core.BuildState) *Client {
	c := &Client{
		state:      state,
		instance:   state.Config.Remote.Instance,
		reqTimeout: time.Duration(state.Config.Remote.Timeout),
		outputs:    map[core.BuildLabel]*pb.Directory{},
	}
	go c.CheckInitialised() // Kick off init now, but we don't have to wait for it.
	return c
}

// CheckInitialised checks that the client has connected to the server correctly.
func (c *Client) CheckInitialised() error {
	c.initOnce.Do(c.init)
	return c.err
}

// init is passed to the sync.Once to do the actual initialisation.
func (c *Client) init() {
	c.err = func() error {
		// Create a copy of the state where we can modify the config
		c.state = c.state.ForConfig()
		c.state.Config.HomeDir = c.state.Config.Remote.HomeDir
		// TODO(peterebden): Add support for TLS.
		client, err := client.NewClient(context.Background(), c.instance, client.DialParams{
			Service:    c.state.Config.Remote.URL,
			CASService: c.state.Config.Remote.CASURL,
			NoSecurity: true,
		}, client.UseBatchOps(true), client.RetryTransient())
		if err != nil {
			return err
		}
		c.client = client
		// Query the server for its capabilities. This tells us whether it is capable of
		// execution, caching or both.
		ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
		defer cancel()
		resp, err := c.client.GetCapabilities(ctx, &pb.GetCapabilitiesRequest{
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
		// Look this up just once now.
		bash, err := core.LookBuildPath("bash", c.state.Config)
		c.bashPath = bash
		c.canBatchBlobReads = c.checkBatchReadBlobs()
		log.Debug("Remote execution client initialised for storage")
		// Now check if it can do remote execution
		if c.state.Config.Remote.NumExecutors > 0 {
			if caps := resp.ExecutionCapabilities; caps != nil {
				if err := c.chooseDigest([]pb.DigestFunction_Value{caps.DigestFunction}); err != nil {
					return err
				} else if !caps.ExecEnabled {
					return fmt.Errorf("Remote execution not enabled for this server")
				}
				c.remoteExecution = true
				c.platform = convertPlatform(c.state.Config)
				log.Debug("Remote execution client initialised for execution")
			} else {
				log.Fatalf("Remote execution is configured but the build server doesn't support it")
			}
		}
		return err
	}()
	if c.err != nil {
		log.Error("Error setting up remote execution client: %s", c.err)
	}
}

// chooseDigest selects a digest function that we will use.w
func (c *Client) chooseDigest(fns []pb.DigestFunction_Value) error {
	systemFn := c.digestEnum(c.state.Config.Build.HashFunction)
	for _, fn := range fns {
		if fn == systemFn {
			return nil
		}
	}
	return fmt.Errorf("No acceptable hash function available; server supports %s but we require %s. Hint: you may need to set the hash function appropriately in the [build] section of your config", fns, systemFn)
}

// digestEnum returns a proto enum for the digest function of given name (as we name them in config)
func (c *Client) digestEnum(name string) pb.DigestFunction_Value {
	switch c.state.Config.Build.HashFunction {
	case "sha256":
		return pb.DigestFunction_SHA256
	case "sha1":
		return pb.DigestFunction_SHA1
	default:
		return pb.DigestFunction_UNKNOWN // Shouldn't get here
	}
}

// Store stores a set of artifacts for a single build target.
func (c *Client) Store(target *core.BuildTarget, metadata *core.BuildMetadata, files []string) error {
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
		for _, filename := range files {
			file := path.Join(outDir, filename)
			info, err := os.Lstat(file)
			if err != nil {
				return err
			} else if mode := info.Mode(); mode&os.ModeDir != 0 {
				// It's a directory, needs special treatment
				root, children, err := c.digestDir(file, nil)
				if err != nil {
					return err
				}
				digest, contents := c.digestMessageContents(&pb.Tree{
					Root:     root,
					Children: children,
				})
				ch <- &blob{
					Digest: digest,
					Data:   contents,
				}
				ar.OutputDirectories = append(ar.OutputDirectories, &pb.OutputDirectory{
					Path:       filename,
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
					Path:   filename,
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
				Path:         filename,
				Digest:       digest,
				IsExecutable: target.IsBinary,
			})
		}
		if len(metadata.Stdout) > 0 {
			h := c.sum(metadata.Stdout)
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
		return nil
	}); err != nil {
		return err
	}
	// OK, now the blobs are uploaded, we also need to upload the Action itself.
	_, digest, err := c.uploadAction(target, false, metadata.Test)
	if err != nil {
		return err
	} else if !metadata.Test {
		if err := c.setOutputs(target.Label, ar); err != nil {
			return err
		}
	}
	// Now we can use that to upload the result itself.
	ctx, cancel := context.WithTimeout(context.Background(), c.reqTimeout)
	defer cancel()
	_, err = c.client.UpdateActionResult(ctx, &pb.UpdateActionResultRequest{
		InstanceName: c.instance,
		ActionDigest: digest,
		ActionResult: ar,
	})
	return err
}

// Retrieve fetches back a set of artifacts for a single build target.
// Its outputs are written out to their final locations.
func (c *Client) Retrieve(target *core.BuildTarget) (*core.BuildMetadata, error) {
	if err := c.CheckInitialised(); err != nil {
		return nil, err
	}
	outDir := target.OutDir()
	if target.IsFilegroup {
		if err := removeOutputs(target); err != nil {
			return nil, err
		}
		return &core.BuildMetadata{}, c.downloadDirectory(outDir, c.targetOutputs(target.Label))
	}
	isTest := target.State() >= core.Built
	needStdout := target.PostBuildFunction != nil && !isTest // We only care in this case.
	inputRoot, err := c.buildInputRoot(target, false, isTest)
	if err != nil {
		return nil, err
	}
	cmd, err := c.buildCommand(target, inputRoot, isTest)
	if err != nil {
		return nil, err
	}
	digest := c.digestMessage(&pb.Action{
		CommandDigest:   c.digestMessage(cmd),
		InputRootDigest: c.digestMessage(inputRoot),
		Timeout:         ptypes.DurationProto(timeout(target, isTest)),
	})
	ctx, cancel := context.WithTimeout(context.Background(), c.reqTimeout)
	defer cancel()
	resp, err := c.client.GetActionResult(ctx, &pb.GetActionResultRequest{
		InstanceName: c.instance,
		ActionDigest: digest,
		InlineStdout: needStdout,
	})
	if err != nil {
		return nil, err
	} else if err := c.setOutputs(target.Label, resp); err != nil {
		return nil, err
	}
	mode := target.OutMode()
	if err := removeOutputs(target); err != nil {
		return nil, err
	}
	if err := c.downloadBlobs(ctx, func(ch chan<- *blob) error {
		defer close(ch)
		for _, file := range resp.OutputFiles {
			filePath := path.Join(outDir, file.Path)
			addPerms := extraPerms(file)
			if file.Contents != nil {
				// Inlining must have been requested. Can write it directly.
				if err := fs.EnsureDir(filePath); err != nil {
					return err
				} else if err := fs.WriteFile(bytes.NewReader(file.Contents), filePath, mode|addPerms); err != nil {
					return err
				}
			} else {
				ch <- &blob{Digest: file.Digest, File: filePath, Mode: mode | addPerms}
			}
		}
		for _, dir := range resp.OutputDirectories {
			dirPath := path.Join(outDir, dir.Path)
			tree := &pb.Tree{}
			if err := c.readByteStreamToProto(dir.TreeDigest, tree); err != nil {
				return err
			}
			if err := c.downloadDirectory(dirPath, tree.Root); err != nil {
				return err
			}
			for _, child := range tree.Children {
				if err := c.downloadDirectory(dirPath, child); err != nil {
					return err
				}
			}
		}
		// For unexplained reasons the protocol treats symlinks differently based on what
		// they point to. We obviously create them in the same way though.
		for _, link := range append(resp.OutputFileSymlinks, resp.OutputDirectorySymlinks...) {
			if err := os.Symlink(link.Target, path.Join(outDir, link.Path)); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, c.wrapActionErr(err, digest)
	}
	return c.buildMetadata(resp, needStdout, false)
}

// Build executes a remote build of the given target.
func (c *Client) Build(tid int, target *core.BuildTarget) (*core.BuildMetadata, error) {
	if err := c.CheckInitialised(); err != nil {
		return nil, err
	}
	if target.IsFilegroup {
		// Filegroups get special-cased since they are just a movement of files.
		return &core.BuildMetadata{}, c.setFilegroupOutputs(target)
	}
	command, digest, err := c.uploadAction(target, true, false)
	if err != nil {
		return nil, err
	}
	metadata, ar, err := c.execute(tid, target, command, digest, target.BuildTimeout, target.PostBuildFunction != nil)
	if err != nil {
		return metadata, err
	}
	return metadata, c.wrapActionErr(c.setOutputs(target.Label, ar), digest)
}

// Test executes a remote test of the given target.
// It returns the results (and coverage if appropriate) as bytes to be parsed elsewhere.
func (c *Client) Test(tid int, target *core.BuildTarget) (metadata *core.BuildMetadata, results, coverage []byte, err error) {
	if err := c.CheckInitialised(); err != nil {
		return nil, nil, nil, err
	}
	command, digest, err := c.uploadAction(target, true, true)
	if err != nil {
		return nil, nil, nil, err
	}
	metadata, ar, execErr := c.execute(tid, target, command, digest, target.TestTimeout, false)
	// Error handling here is a bit fiddly due to prioritisation; the execution error
	// is more relevant, but we want to still try to get results if we can, and it's an
	// error if we can't get those results on success.
	if !target.NoTestOutput && ar != nil {
		results, err = c.readAllByteStream(c.digestForFilename(ar, core.TestResultsFile))
		if execErr == nil && err != nil {
			return metadata, nil, nil, err
		}
	}
	if target.NeedCoverage(c.state) && ar != nil {
		coverage, err = c.readAllByteStream(c.digestForFilename(ar, core.CoverageFile))
		if execErr == nil && err != nil {
			return metadata, results, nil, err
		}
	}
	return metadata, results, coverage, execErr
}

// execute submits an action to the remote executor and monitors its progress.
// The returned ActionResult may be nil on failure.
func (c *Client) execute(tid int, target *core.BuildTarget, command *pb.Command, digest *pb.Digest, timeout time.Duration, needStdout bool) (*core.BuildMetadata, *pb.ActionResult, error) {
	// First see if this execution is cached
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if ar, err := c.client.GetActionResult(ctx, &pb.GetActionResultRequest{
		InstanceName: c.instance,
		ActionDigest: digest,
		InlineStdout: needStdout,
	}); err == nil {
		// This action already exists and has been cached.
		if metadata, err := c.buildMetadata(ar, needStdout, false); err == nil {
			log.Debug("Got remotely cached results for %s %s", target.Label, c.actionURL(digest, true))
			return metadata, ar, c.verifyActionResult(target, command, digest, ar)
		}
	}
	ctx, cancel = context.WithTimeout(context.Background(), timeout)
	defer cancel()
	stream, err := c.client.Execute(ctx, &pb.ExecuteRequest{
		InstanceName: c.instance,
		ActionDigest: digest,
	})
	if err != nil {
		return nil, nil, err
	}
	for {
		resp, err := stream.Recv()
		if err != nil {
			// We shouldn't get an EOF here because there's in-channel signalling via Done.
			// TODO(peterebden): On other errors we should be able to reconnect and use
			//                   WaitExecution to rejoin the stream.
			return nil, nil, err
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
				return nil, nil, convertError(result.Error)
			case *longrunning.Operation_Response:
				response := &pb.ExecuteResponse{}
				if err := ptypes.UnmarshalAny(result.Response, response); err != nil {
					log.Error("Failed to deserialise execution response: %s", err)
					return nil, nil, err
				}
				if response.CachedResult {
					c.state.LogBuildResult(tid, target.Label, core.TargetCached, "Cached")
				}
				for k, v := range response.ServerLogs {
					log.Debug("Server log available: %s: hash key %s", k, v.Digest.Hash)
				}
				var respErr error
				if response.Status != nil {
					respErr = convertError(response.Status)
				}
				if resp.Result == nil { // This is optional on failure.
					return nil, nil, respErr
				}
				if response.Result == nil { // This seems to happen when things go wrong on the build server end.
					if response.Status != nil {
						return nil, nil, fmt.Errorf("Build server returned invalid result: %s", convertError(response.Status))
					}
					log.Debug("Bad result from build server: %+v", response)
					return nil, nil, fmt.Errorf("Build server did not return valid result")
				}
				if response.Message != "" {
					// Informational messages can be emitted on successful actions.
					log.Debug("Message from build server:\n     %s", response.Message)
				}
				failed := respErr != nil || response.Result.ExitCode != 0
				metadata, err := c.buildMetadata(response.Result, needStdout || failed, failed)
				// The original error is higher priority than us trying to retrieve the
				// output of the thing that failed.
				if respErr != nil {
					return metadata, response.Result, respErr
				} else if response.Result.ExitCode != 0 {
					err := fmt.Errorf("Remotely executed command exited with %d", response.Result.ExitCode)
					if response.Message != "" {
						err = fmt.Errorf("%s\n    %s", err, response.Message)
					}
					if len(metadata.Stdout) != 0 {
						err = fmt.Errorf("%s\nStdout:\n%s", err, metadata.Stdout)
					}
					if len(metadata.Stderr) != 0 {
						err = fmt.Errorf("%s\nStderr:\n%s", err, metadata.Stderr)
					}
					return nil, nil, err
				} else if err != nil {
					return nil, nil, err
				}
				return metadata, response.Result, c.verifyActionResult(target, command, digest, response.Result)
			}
		}
	}
}

// updateProgress updates the progress of a target based on its metadata.
func (c *Client) updateProgress(tid int, target *core.BuildTarget, metadata *pb.ExecuteOperationMetadata) {
	if c.state.Config.Remote.DisplayURL != "" {
		log.Debug("Remote progress for %s: %s%s", target.Label, metadata.Stage, c.actionURL(metadata.ActionDigest, true))
	}
	if target.State() <= core.Built {
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

// PrintHashes prints the action hashes for a target.
func (c *Client) PrintHashes(target *core.BuildTarget, isTest bool) {
	inputRoot, err := c.buildInputRoot(target, false, isTest)
	if err != nil {
		log.Fatalf("Unable to calculate input hash: %s", err)
	}
	fmt.Printf("Remote execution hashes:\n")
	inputRootDigest := c.digestMessage(inputRoot)
	fmt.Printf("  Input: %7d bytes: %s\n", inputRootDigest.SizeBytes, inputRootDigest.Hash)
	cmd, _ := c.buildCommand(target, inputRoot, isTest)
	commandDigest := c.digestMessage(cmd)
	fmt.Printf("Command: %7d bytes: %s\n", commandDigest.SizeBytes, commandDigest.Hash)
	if c.state.Config.Remote.DisplayURL != "" {
		fmt.Printf("    URL: %s\n", c.actionURL(commandDigest, false))
	}
	actionDigest := c.digestMessage(&pb.Action{
		CommandDigest:   commandDigest,
		InputRootDigest: inputRootDigest,
		Timeout:         ptypes.DurationProto(timeout(target, isTest)),
	})
	fmt.Printf(" Action: %7d bytes: %s\n", actionDigest.SizeBytes, actionDigest.Hash)
	if c.state.Config.Remote.DisplayURL != "" {
		fmt.Printf("    URL: %s\n", c.actionURL(actionDigest, false))
	}
}
