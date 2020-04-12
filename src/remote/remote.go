// Package remote provides our interface to the Google remote execution APIs
// (https://github.com/bazelbuild/remote-apis) which Please can use to distribute
// work to remote servers.
package remote

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/bazelbuild/remote-apis-sdks/go/pkg/chunker"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/client"
	sdkdigest "github.com/bazelbuild/remote-apis-sdks/go/pkg/digest"
	fpb "github.com/bazelbuild/remote-apis/build/bazel/remote/asset/v1"
	pb "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/bazelbuild/remote-apis/build/bazel/semver"
	"github.com/golang/protobuf/ptypes"
	"github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"golang.org/x/sync/errgroup"
	"google.golang.org/genproto/googleapis/longrunning"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	_ "google.golang.org/grpc/encoding/gzip" // Registers the gzip compressor at init
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/status"
	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/core"
)

var log = logging.MustGetLogger("remote")

// Timeout to initially contact the server.
const dialTimeout = 5 * time.Second

// The API version we support.
var apiVersion = semver.SemVer{Major: 2}

// A Client is the interface to the remote API.
//
// It provides a higher-level interface over the specific RPCs available.
type Client struct {
	client      *client.Client
	fetchClient fpb.FetchClient
	initOnce    sync.Once
	state       *core.BuildState
	reqTimeout  time.Duration
	err         error // for initialisation
	instance    string

	// Stored output directories from previously executed targets.
	// This isn't just a cache - it is needed for cases where we don't actually
	// have the files physically on disk.
	outputs     map[core.BuildLabel]*pb.Directory
	outputMutex sync.RWMutex

	// Used to control downloading targets (we must make sure we don't re-fetch them
	// while another target is trying to use them).
	downloads sync.Map

	// Server-sent cache properties
	maxBlobBatchSize int64
	cacheWritable    bool

	// Platform properties that we will request from the remote.
	// TODO(peterebden): this will need some modification for cross-compiling support.
	platform *pb.Platform

	// Cache this for later
	bashPath string

	// Stats used to report RPC data rates
	byteRateIn, byteRateOut, totalBytesIn, totalBytesOut int
	stats                                                *statsHandler
}

// A pendingDownload represents a pending download of a build target. It is used to
// ensure we only download each target exactly once.
type pendingDownload struct {
	once sync.Once
	err  error // Any error if the download failed.
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
	c.stats = newStatsHandler(c)
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
	// Disable all logging from glog (which is transitively called from remote-apis-sdks)
	flag.CommandLine.Parse([]string{"-v", "0", "-log_dir", "/dev/null"})
	// Change grpc to log using our implementation
	grpclog.SetLoggerV2(&grpcLogMabob{})
	var g errgroup.Group
	g.Go(c.initExec)
	g.Go(c.initFetch)
	c.err = g.Wait()
	if c.err != nil {
		log.Error("Error setting up remote execution client: %s", c.err)
	}
}

// initExec initialiases the remote execution client.
func (c *Client) initExec() error {
	// Create a copy of the state where we can modify the config
	c.state = c.state.ForConfig()
	c.state.Config.HomeDir = c.state.Config.Remote.HomeDir
	client, err := client.NewClient(context.Background(), c.instance, client.DialParams{
		Service:            c.state.Config.Remote.URL,
		CASService:         c.state.Config.Remote.CASURL,
		NoSecurity:         !c.state.Config.Remote.Secure,
		TransportCredsOnly: c.state.Config.Remote.Secure,
		DialOpts: []grpc.DialOption{
			grpc.WithStatsHandler(c.stats),
			// Set an arbitrarily large (400MB) max message size so it isn't a limitation.
			grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(419430400)),
		},
	}, client.UseBatchOps(true), client.RetryTransient())
	if err != nil {
		return err
	}
	c.client = client
	// Query the server for its capabilities. This tells us whether it is capable of
	// execution, caching or both.
	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()
	resp, err := c.client.GetCapabilities(ctx)
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
	log.Debug("Remote execution client initialised for storage")
	// Now check if it can do remote execution
	if resp.ExecutionCapabilities == nil {
		return fmt.Errorf("Remote execution is configured but the build server doesn't support it")
	}
	if err := c.chooseDigest([]pb.DigestFunction_Value{resp.ExecutionCapabilities.DigestFunction}); err != nil {
		return err
	} else if !resp.ExecutionCapabilities.ExecEnabled {
		return fmt.Errorf("Remote execution not enabled for this server")
	}
	c.platform = convertPlatform(c.state.Config)
	log.Debug("Remote execution client initialised for execution")
	return nil
}

// initFetch initialises the remote fetch server.
func (c *Client) initFetch() error {
	if c.state.Config.Remote.AssetURL == "" {
		return fmt.Errorf("You must specify remote.asseturl in configuration to use remote execution")
	}
	tlsOption := func() grpc.DialOption {
		if c.state.Config.Remote.Secure {
			return grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, ""))
		}
		return grpc.WithInsecure()
	}
	conn, err := grpc.Dial(c.state.Config.Remote.AssetURL,
		grpc.WithUnaryInterceptor(grpc_retry.UnaryClientInterceptor()),
		tlsOption(),
	)
	if err != nil {
		return fmt.Errorf("Failed to connect to the remote fetch server: %s", err)
	}
	c.fetchClient = fpb.NewFetchClient(conn)
	return nil
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

// Build executes a remote build of the given target.
func (c *Client) Build(tid int, target *core.BuildTarget) (*core.BuildMetadata, error) {
	if err := c.CheckInitialised(); err != nil {
		return nil, err
	}
	metadata, ar, digest, err := c.build(tid, target)
	if err != nil {
		return metadata, err
	}
	hash, _ := hex.DecodeString(c.digestMessage(ar).Hash)
	if c.state.TargetHasher != nil {
		c.state.TargetHasher.SetHash(target, hash)
	}
	if err := c.setOutputs(target.Label, ar); err != nil {
		return metadata, c.wrapActionErr(err, digest)
	}
	// Need to download the target if it was originally requested (and the user didn't pass --nodownload).
	// Also anything needed for subinclude needs to be local.
	if (c.state.IsOriginalTarget(target.Label) && c.state.DownloadOutputs && !c.state.NeedTests) || target.NeededForSubinclude {
		if !c.outputsExist(target, digest) {
			c.state.LogBuildResult(tid, target.Label, core.TargetBuilding, "Downloading")
			if err := c.download(target, func() error {
				return c.reallyDownload(target, digest, ar)
			}); err != nil {
				return metadata, err
			}
		}
	}
	return metadata, nil
}

// build implements the actual build of a target.
func (c *Client) build(tid int, target *core.BuildTarget) (*core.BuildMetadata, *pb.ActionResult, *pb.Digest, error) {
	needStdout := target.PostBuildFunction != nil
	// If we're gonna stamp the target, first check the unstamped equivalent that we store results under.
	// This implements the rules of stamp whereby we don't force rebuilds every time e.g. the SCM revision changes.
	command, stampedDigest, unstampedDigest, err := c.buildStampedAndUnstampedAction(target)
	if err != nil {
		return nil, nil, nil, err
	}
	if target.Stamp {
		if metadata, ar := c.maybeRetrieveResults(tid, target, command, unstampedDigest, needStdout); metadata != nil {
			return metadata, ar, stampedDigest, nil
		}
	}
	metadata, ar, err := c.execute(tid, target, command, stampedDigest, target.BuildTimeout, false, needStdout)
	if target.Stamp && err == nil {
		// Store results under unstamped digest too.
		c.locallyCacheResults(target, unstampedDigest, metadata, ar)
		ctx, cancel := context.WithTimeout(context.Background(), c.reqTimeout)
		defer cancel()
		c.client.UpdateActionResult(ctx, &pb.UpdateActionResultRequest{
			InstanceName: c.instance,
			ActionDigest: unstampedDigest,
			ActionResult: ar,
		})
	}
	return metadata, ar, stampedDigest, err
}

// Download downloads outputs for the given target.
func (c *Client) Download(target *core.BuildTarget) error {
	return c.download(target, func() error {
		command, digest, err := c.buildAction(target, false, target.Stamp)
		if err != nil {
			return fmt.Errorf("Failed to create action for %s: %s", target, err)
		}
		_, ar := c.retrieveResults(target, command, digest, false)
		if ar == nil {
			return fmt.Errorf("Failed to retrieve action result for %s", target)
		} else if c.outputsExist(target, digest) {
			return nil
		}
		return c.reallyDownload(target, digest, ar)
	})
}

func (c *Client) download(target *core.BuildTarget, f func() error) error {
	v, _ := c.downloads.LoadOrStore(target, &pendingDownload{})
	d := v.(*pendingDownload)
	d.once.Do(func() {
		d.err = f()
	})
	return d.err
}

func (c *Client) reallyDownload(target *core.BuildTarget, digest *pb.Digest, ar *pb.ActionResult) error {
	log.Debug("Downloading outputs for %s", target)
	if err := removeOutputs(target); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), c.reqTimeout)
	defer cancel()
	if err := c.client.DownloadActionOutputs(ctx, ar, target.OutDir()); err != nil {
		return c.wrapActionErr(err, digest)
	}
	c.recordAttrs(target, digest)
	log.Debug("Downloaded outputs for %s", target)
	return nil
}

// Test executes a remote test of the given target.
// It returns the results (and coverage if appropriate) as bytes to be parsed elsewhere.
func (c *Client) Test(tid int, target *core.BuildTarget) (metadata *core.BuildMetadata, results [][]byte, coverage []byte, err error) {
	if err := c.CheckInitialised(); err != nil {
		return nil, nil, nil, err
	}
	command, digest, err := c.buildAction(target, true, false)
	if err != nil {
		return nil, nil, nil, err
	}
	metadata, ar, execErr := c.execute(tid, target, command, digest, target.TestTimeout, true, false)
	// Error handling here is a bit fiddly due to prioritisation; the execution error
	// is more relevant, but we want to still try to get results if we can, and it's an
	// error if we can't get those results on success.
	if !target.NoTestOutput && ar != nil {
		results, err = c.downloadAllPrefixedFiles(ar, core.TestResultsFile)
		if execErr == nil && err != nil {
			return metadata, nil, nil, err
		}
	}
	if target.NeedCoverage(c.state) && ar != nil {
		if digest := c.digestForFilename(ar, core.CoverageFile); digest != nil {
			ctx, cancel := context.WithTimeout(context.Background(), c.reqTimeout)
			defer cancel()
			coverage, err = c.client.ReadBlob(ctx, sdkdigest.NewFromProtoUnvalidated(digest))
			if execErr == nil && err != nil {
				return metadata, results, nil, err
			}
		}
	}
	return metadata, results, coverage, execErr
}

// retrieveResults retrieves target results from where it can (either from the local cache or from remote).
// It returns nil if it cannot be retrieved.
func (c *Client) retrieveResults(target *core.BuildTarget, command *pb.Command, digest *pb.Digest, needStdout bool) (*core.BuildMetadata, *pb.ActionResult) {
	// First see if this execution is cached locally
	if metadata, ar := c.retrieveLocalResults(target, digest); metadata != nil {
		log.Debug("Got locally cached results for %s %s", target.Label, c.actionURL(digest, true))
		return metadata, ar
	}
	// Now see if it is cached on the remote server
	ctx, cancel := context.WithTimeout(context.Background(), c.reqTimeout)
	defer cancel()
	if ar, err := c.client.GetActionResult(ctx, &pb.GetActionResultRequest{
		InstanceName: c.instance,
		ActionDigest: digest,
		InlineStdout: needStdout,
	}); err == nil {
		// This action already exists and has been cached.
		if metadata, err := c.buildMetadata(ar, needStdout, false); err == nil {
			log.Debug("Got remotely cached results for %s %s", target.Label, c.actionURL(digest, true))
			err := c.verifyActionResult(target, command, digest, ar, c.state.Config.Remote.VerifyOutputs)
			if err == nil {
				c.locallyCacheResults(target, digest, metadata, ar)
				return metadata, ar
			}
			log.Debug("Remotely cached results for %s were missing some outputs, forcing a rebuild: %s", target.Label, err)
		}
	}
	return nil, nil
}

// maybeRetrieveResults is like retrieveResults but only retrieves if we aren't forcing a rebuild of the target
// (i.e. not if we're doing plz build --rebuild).
func (c *Client) maybeRetrieveResults(tid int, target *core.BuildTarget, command *pb.Command, digest *pb.Digest, needStdout bool) (*core.BuildMetadata, *pb.ActionResult) {
	if !(c.state.ForceRebuild && (c.state.IsOriginalTarget(target.Label) || c.state.IsOriginalTarget(target.Label.Parent()))) {
		c.state.LogBuildResult(tid, target.Label, core.TargetBuilding, "Checking remote...")
		if metadata, ar := c.retrieveResults(target, command, digest, needStdout); metadata != nil {
			return metadata, ar
		}
	}
	return nil, nil
}

// execute submits an action to the remote executor and monitors its progress.
// The returned ActionResult may be nil on failure.
func (c *Client) execute(tid int, target *core.BuildTarget, command *pb.Command, digest *pb.Digest, timeout time.Duration, isTest, needStdout bool) (*core.BuildMetadata, *pb.ActionResult, error) {
	if metadata, ar := c.maybeRetrieveResults(tid, target, command, digest, needStdout); metadata != nil {
		return metadata, ar, nil
	}
	// We didn't actually upload the inputs before, so we must do so now.
	command, digest, err := c.uploadAction(target, isTest)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to upload build action: %s", err)
	}
	// Remote actions & filegroups get special treatment at this point.
	if target.IsFilegroup {
		// Filegroups get special-cased since they are just a movement of files.
		return c.buildFilegroup(target, command, digest)
	} else if target.IsRemoteFile {
		return c.fetchRemoteFile(tid, target, digest)
	}
	if timeout < c.reqTimeout {
		// This is the timeout for the individual action to execute, but doesn't necessarily
		// take into account time to fetch inputs etc, so we might need to extend.
		timeout = c.reqTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	resp, err := c.client.ExecuteAndWaitProgress(ctx, &pb.ExecuteRequest{
		InstanceName:    c.instance,
		ActionDigest:    digest,
		SkipCacheLookup: true, // We've already done it above.
	}, func(metadata *pb.ExecuteOperationMetadata) {
		c.updateProgress(tid, target, metadata)
	})
	if err != nil {
		// Handle timing issues if we try to resume an execution as it fails. If we get a
		// "not found" we might find that it's already been completed and we can't resume.
		if status.Code(err) == codes.NotFound {
			if metadata, ar := c.retrieveResults(target, command, digest, needStdout); metadata != nil {
				return metadata, ar, nil
			}
		}
		return nil, nil, c.wrapActionErr(fmt.Errorf("Failed to execute %s: %s", target, err), digest)
	}
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
			if respErr != nil {
				if !strings.Contains(respErr.Error(), c.state.Config.Remote.DisplayURL) {
					if url := c.actionURL(digest, false); url != "" {
						respErr = fmt.Errorf("%s\nAction URL: %s", respErr, url)
					}
				}
			}
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
			// Add a link to the action URL, but only if the server didn't do it (they
			// might add one to the failed action if they're using the Buildbarn extension
			// for it, which we can't replicate here).
			if !strings.Contains(response.Message, c.state.Config.Remote.DisplayURL) {
				if url := c.actionURL(digest, true); url != "" {
					err = fmt.Errorf("%s\n%s", err, url)
				}
			}
			return nil, nil, err
		} else if err != nil {
			return nil, nil, err
		}
		log.Debug("Completed remote build action for %s; input fetch %s, build time %s", target, metadata.InputFetchEndTime.Sub(metadata.InputFetchStartTime), metadata.EndTime.Sub(metadata.StartTime))
		if err := c.verifyActionResult(target, command, digest, response.Result, false); err != nil {
			return metadata, response.Result, err
		}
		c.locallyCacheResults(target, digest, metadata, response.Result)
		return metadata, response.Result, nil
	default:
		if !resp.Done {
			return nil, nil, fmt.Errorf("Received an incomplete response for %s", target)
		}
		return nil, nil, fmt.Errorf("Unknown response type (was a %T): %#v", resp.Result, resp) // Shouldn't get here
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
			c.state.LogBuildResult(tid, target.Label, core.TargetBuilding, "Checking cache...")
		case pb.ExecutionStage_QUEUED:
			c.state.LogBuildResult(tid, target.Label, core.TargetBuilding, "Queued")
		case pb.ExecutionStage_EXECUTING:
			c.state.LogBuildResult(tid, target.Label, core.TargetBuilding, "Building...")
		case pb.ExecutionStage_COMPLETED:
			c.state.LogBuildResult(tid, target.Label, core.TargetBuilt, "Built")
		}
	} else {
		switch metadata.Stage {
		case pb.ExecutionStage_CACHE_CHECK:
			c.state.LogBuildResult(tid, target.Label, core.TargetTesting, "Checking cache...")
		case pb.ExecutionStage_QUEUED:
			c.state.LogBuildResult(tid, target.Label, core.TargetTesting, "Queued")
		case pb.ExecutionStage_EXECUTING:
			c.state.LogBuildResult(tid, target.Label, core.TargetTesting, "Testing...")
		case pb.ExecutionStage_COMPLETED:
			c.state.LogBuildResult(tid, target.Label, core.TargetTested, "Tested")
		}
	}
}

// PrintHashes prints the action hashes for a target.
func (c *Client) PrintHashes(target *core.BuildTarget, isTest bool) {
	inputRoot, err := c.uploadInputs(nil, target, isTest)
	if err != nil {
		log.Fatalf("Unable to calculate input hash: %s", err)
	}
	fmt.Printf("Remote execution hashes:\n")
	inputRootDigest := c.digestMessage(inputRoot)
	fmt.Printf("  Input: %7d bytes: %s\n", inputRootDigest.SizeBytes, inputRootDigest.Hash)
	cmd, _ := c.buildCommand(target, inputRoot, isTest, target.Stamp)
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

// DataRate returns an estimate of the current in/out RPC data rates in bytes per second.
func (c *Client) DataRate() (int, int, int, int) {
	return c.byteRateIn, c.byteRateOut, c.totalBytesIn, c.totalBytesOut
}

// fetchRemoteFile sends a request to fetch a file using the remote asset API.
func (c *Client) fetchRemoteFile(tid int, target *core.BuildTarget, actionDigest *pb.Digest) (*core.BuildMetadata, *pb.ActionResult, error) {
	c.state.LogBuildResult(tid, target.Label, core.TargetBuilding, "Downloading...")
	urls := target.AllURLs(c.state.Config)
	req := &fpb.FetchBlobRequest{
		InstanceName: c.instance,
		Timeout:      ptypes.DurationProto(target.BuildTimeout),
		Uris:         urls,
	}
	if sri := subresourceIntegrity(target); sri != "" {
		req.Qualifiers = []*fpb.Qualifier{{
			Name:  "checksum.sri",
			Value: sri,
		}}
	}
	ctx, cancel := context.WithTimeout(context.Background(), target.BuildTimeout)
	defer cancel()
	resp, err := c.fetchClient.FetchBlob(ctx, req)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to download file: %s", err)
	}
	c.state.LogBuildResult(tid, target.Label, core.TargetBuilt, "Downloaded.")
	// If we get here, the blob exists in the CAS. Create an ActionResult corresponding to it.
	outs := target.Outputs()
	ar := &pb.ActionResult{
		OutputFiles: []*pb.OutputFile{{
			Path:         outs[0],
			Digest:       resp.BlobDigest,
			IsExecutable: target.IsBinary,
		}},
	}
	ctx, cancel = context.WithTimeout(context.Background(), c.reqTimeout)
	defer cancel()
	if _, err := c.client.UpdateActionResult(ctx, &pb.UpdateActionResultRequest{
		InstanceName: c.instance,
		ActionDigest: actionDigest,
		ActionResult: ar,
	}); err != nil {
		return nil, nil, fmt.Errorf("Error updating action result: %s", err)
	}
	return &core.BuildMetadata{}, ar, nil
}

// buildFilegroup "builds" a single filegroup target.
func (c *Client) buildFilegroup(target *core.BuildTarget, command *pb.Command, actionDigest *pb.Digest) (*core.BuildMetadata, *pb.ActionResult, error) {
	b, err := c.uploadInputDir(nil, target, false) // We don't need to actually upload the inputs here, that is already done.
	if err != nil {
		return nil, nil, err
	}
	ar := &pb.ActionResult{}
	if err := c.uploadBlobs(func(ch chan<- *chunker.Chunker) error {
		defer close(ch)
		b.Root(ch)
		for _, out := range command.OutputPaths {
			if d, f := b.Node(path.Join(target.Label.PackageName, out)); d != nil {
				chomk, _ := chunker.NewFromProto(b.Tree(path.Join(target.Label.PackageName, out)), int(c.client.ChunkMaxSize))
				ch <- chomk
				ar.OutputDirectories = append(ar.OutputDirectories, &pb.OutputDirectory{
					Path:       out,
					TreeDigest: chomk.Digest().ToProto(),
				})
			} else if f != nil {
				if target.IsHashFilegroup {
					out = updateHashFilename(out, f.Digest)
				}
				ar.OutputFiles = append(ar.OutputFiles, &pb.OutputFile{
					Path:         out,
					Digest:       f.Digest,
					IsExecutable: f.IsExecutable,
				})
			} else {
				// Of course, we should not get here (classic developer things...)
				return fmt.Errorf("Missing output from filegroup: %s", out)
			}
		}
		return nil
	}); err != nil {
		return nil, nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), c.reqTimeout)
	defer cancel()
	if _, err := c.client.UpdateActionResult(ctx, &pb.UpdateActionResultRequest{
		InstanceName: c.instance,
		ActionDigest: actionDigest,
		ActionResult: ar,
	}); err != nil {
		return nil, nil, fmt.Errorf("Error updating action result: %s", err)
	}
	return &core.BuildMetadata{}, ar, nil
}

// A grpcLogMabob is an implementation of grpc's logging interface using our backend.
type grpcLogMabob struct{}

func (g *grpcLogMabob) Info(args ...interface{})                    { log.Info("%s", args) }
func (g *grpcLogMabob) Infof(format string, args ...interface{})    { log.Info(format, args...) }
func (g *grpcLogMabob) Infoln(args ...interface{})                  { log.Info("%s", args) }
func (g *grpcLogMabob) Warning(args ...interface{})                 { log.Warning("%s", args) }
func (g *grpcLogMabob) Warningf(format string, args ...interface{}) { log.Warning(format, args...) }
func (g *grpcLogMabob) Warningln(args ...interface{})               { log.Warning("%s", args) }
func (g *grpcLogMabob) Error(args ...interface{})                   { log.Error("", args...) }
func (g *grpcLogMabob) Errorf(format string, args ...interface{})   { log.Errorf(format, args...) }
func (g *grpcLogMabob) Errorln(args ...interface{})                 { log.Error("", args...) }
func (g *grpcLogMabob) Fatal(args ...interface{})                   { log.Fatal(args...) }
func (g *grpcLogMabob) Fatalf(format string, args ...interface{})   { log.Fatalf(format, args...) }
func (g *grpcLogMabob) Fatalln(args ...interface{})                 { log.Fatal(args...) }
func (g *grpcLogMabob) V(l int) bool                                { return log.IsEnabledFor(logging.Level(l)) }
