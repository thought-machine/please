// Package remote provides our interface to the Google remote execution APIs
// (https://github.com/bazelbuild/remote-apis) which Please can use to distribute
// work to remote servers.
package remote

import (
	"context"
	"encoding/hex"
	"fmt"
	iofs "io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/client"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/digest"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/filemetadata"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/retry"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/uploadinfo"
	fpb "github.com/bazelbuild/remote-apis/build/bazel/remote/asset/v1"
	pb "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/bazelbuild/remote-apis/build/bazel/semver"
	"github.com/google/uuid"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/metrics"
	remotefs "github.com/thought-machine/please/src/remote/fs"
	"github.com/thought-machine/please/src/remote/fs/cache"
)

var log = logging.Log

// The API version we support.
var apiVersion = semver.SemVer{Major: 2}

var remoteCacheReadDuration = metrics.NewHistogramVec(
	"remote",
	"cache_read_duration",
	"Time taken to read the remote cache, in milliseconds",
	metrics.ExponentialBuckets(0.1, 2, 12), // 12 buckets, starting at 0.1ms and doubling in width.
	[]string{"ci"},
)

// A Client is the interface to the remote API.
//
// It provides a higher-level interface over the specific RPCs available.
type Client struct {
	client         *client.Client
	remoteFSClient remotefs.Client
	fetchClient    fpb.FetchClient
	initOnce       sync.Once
	state          *core.BuildState
	err            error // for initialisation
	instance       string

	// Stored output directories from previously executed targets.
	// This isn't just a cache - it is needed for cases where we don't actually
	// have the files physically on disk.
	outputs map[core.BuildLabel]*pb.Directory
	// subrepoTrees is used to cache the full output tree of a subrepo so we can parse and build those targets without
	// downloading the subrepo locally.
	subrepoTrees map[core.BuildLabel]*pb.Tree
	outputMutex  sync.RWMutex

	// The unstamped build action digests. Stamped and test digests are not stored.
	// This isn't just a cache - it is needed because building a target can modify the target and things like plz hash
	// --detailed and --shell will fail to get the right action digest.
	unstampedBuildActionDigests actionDigestMap

	// Used to control downloading targets (we must make sure we don't re-fetch them
	// while another target is trying to use them).
	//
	// This map is of effective type `map[*core.BuildTarget]*pendingDownload`
	downloads sync.Map

	// Used to store directories output from actions.
	//
	// This map is of effective type `map[string]*pb.Directory`
	directories sync.Map

	// Server-sent cache properties
	maxBlobBatchSize int64

	// Platform properties that we will request from the remote.
	// TODO(peterebden): this will need some modification for cross-compiling support.
	platform *pb.Platform

	// Path to the shell to use to execute actions in.
	shellPath string

	// User's home directory.
	userHome string

	// Remote build ID
	buildID string

	// Stats used to report RPC data rates
	stats *statsHandler

	// Used to store and retrieve action results to reduce RPC calls when re-building targets
	mdStore buildMetadataStore

	// Passed to various SDK functions.
	fileMetadataCache filemetadata.Cache

	// existingBlobs is used to track the set of existing blobs remotely.
	existingBlobs     map[string]struct{}
	existingBlobMutex sync.Mutex
}

type actionDigestMap struct {
	m sync.Map
}

func (m *actionDigestMap) Get(label core.BuildLabel) *pb.Digest {
	d, ok := m.m.Load(label)
	if !ok {
		panic("could not find action digest for label: " + label.String())
	}
	return d.(*pb.Digest)
}

func (m *actionDigestMap) Put(label core.BuildLabel, actionDigest *pb.Digest) {
	m.m.Store(label, actionDigest)
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
		state:        state,
		instance:     state.Config.Remote.Instance,
		outputs:      make(map[core.BuildLabel]*pb.Directory, 100),
		subrepoTrees: make(map[core.BuildLabel]*pb.Tree, 10),
		mdStore:      newDirMDStore(state.Config.Remote.Instance, time.Duration(state.Config.Remote.CacheDuration)),
		existingBlobs: map[string]struct{}{
			digest.Empty.Hash: {},
		},
		fileMetadataCache: filemetadata.NewNoopCache(),
		shellPath:         state.Config.Remote.Shell,
		buildID:           state.Config.Remote.BuildID,
		stats:             newStatsHandler(),
	}
	go c.CheckInitialised() // Kick off init now, but we don't have to wait for it.
	return c
}

// CheckInitialised checks that the client has connected to the server correctly.
func (c *Client) CheckInitialised() error {
	c.initOnce.Do(c.init)
	return c.err
}

// Disconnect disconnects this client from the remote server.
func (c *Client) Disconnect() error {
	if c.client != nil {
		log.Debug("Disconnecting from remote execution server...")
		return c.client.Close()
	}
	return nil
}

// init is passed to the sync.Once to do the actual initialisation.
func (c *Client) init() {
	// Change grpc to log using our implementation
	grpclog.SetLoggerV2(&grpcLogMabob{})
	var g errgroup.Group
	g.Go(c.initExec)
	if c.state.Config.Remote.AssetURL != "" {
		g.Go(c.initFetch)
	}
	c.err = g.Wait()
	if c.err != nil {
		log.Error("Error setting up remote execution client: %s", c.err)
	}
}

// initExec initialiases the remote execution client.
func (c *Client) initExec() error {
	if c.buildID == "" {
		id, _ := uuid.NewRandom()
		c.buildID = id.String()
	}
	// Create a copy of the state where we can modify the config
	dialOpts, err := c.dialOpts()
	if err != nil {
		return err
	}
	client, err := client.NewClient(context.Background(), c.instance, client.DialParams{
		Service:            c.state.Config.Remote.URL,
		CASService:         c.state.Config.Remote.CASURL,
		NoSecurity:         !c.state.Config.Remote.Secure,
		TransportCredsOnly: c.state.Config.Remote.Secure,
		DialOpts:           dialOpts,
	}, client.UseBatchOps(true), &client.TreeSymlinkOpts{Preserved: true}, client.RetryTransient(), client.RPCTimeouts(map[string]time.Duration{
		"default":          time.Duration(c.state.Config.Remote.Timeout),
		"GetCapabilities":  5 * time.Second,
		"BatchUpdateBlobs": time.Minute,
		"BatchReadBlobs":   time.Minute,
		"GetTree":          time.Minute,
		"Execute":          0,
		"WaitExecution":    0,
	}))
	if err != nil {
		return err
	}
	c.client = client
	if c.state.Config.Cache.Dir != "" {
		c.remoteFSClient = cache.New(c.client, filepath.Join(c.state.Config.Cache.Dir, "cas-cache"))
	} else {
		c.remoteFSClient = client
	}
	// Extend timeouts a bit, RetryTransient only gives about 1.5 seconds total which isn't
	// necessarily very much if the other end needs to sort its life out.
	c.client.Retrier.Backoff = retry.ExponentialBackoff(500*time.Millisecond, 5*time.Second, retry.Attempts(8))
	// Query the server for its capabilities. This tells us whether it is capable of
	// execution, caching or both.
	resp, err := c.client.GetCapabilities(context.Background())
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
	if err := c.chooseDigest(caps.DigestFunctions); err != nil {
		return err
	}
	c.maxBlobBatchSize = caps.MaxBatchTotalSizeBytes
	if c.maxBlobBatchSize == 0 {
		// No limit was set by the server, assume we are implicitly limited to 4MB (that's
		// gRPC's limit which most implementations do not seem to override). Round it down a
		// bit to allow a bit of serialisation overhead etc.
		c.maxBlobBatchSize = 4000000
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("Failed to determine user home dir: %s", err)
	}
	c.userHome = home

	// Now check if it can do remote execution
	if resp.ExecutionCapabilities == nil {
		return fmt.Errorf("Remote execution is configured but the build server doesn't support it")
	}
	if err := c.chooseDigest([]pb.DigestFunction_Value{resp.ExecutionCapabilities.DigestFunction}); err != nil {
		return err
	} else if !resp.ExecutionCapabilities.ExecEnabled {
		return fmt.Errorf("Remote execution not enabled for this server")
	}
	c.platform = convertPlatform(c.state.Config.Remote.Platform)
	log.Debug("Remote execution client initialised")
	if c.state.Config.Remote.AssetURL == "" {
		c.fetchClient = fpb.NewFetchClient(client.Connection)
	}
	return nil
}

// initFetch initialises the remote fetch server.
func (c *Client) initFetch() error {
	dialOpts, err := c.dialOpts()
	if err != nil {
		return err
	}
	if c.state.Config.Remote.Secure {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	conn, err := grpc.Dial(c.state.Config.Remote.AssetURL, append(dialOpts, grpc.WithUnaryInterceptor(grpc_retry.UnaryClientInterceptor()))...)
	if err != nil {
		return fmt.Errorf("Failed to connect to the remote fetch server: %s", err)
	}
	c.fetchClient = fpb.NewFetchClient(conn)
	return nil
}

// chooseDigest selects a digest function that we will use.w
func (c *Client) chooseDigest(fns []pb.DigestFunction_Value) error {
	systemFn := c.digestEnum()
	for _, fn := range fns {
		if fn == systemFn {
			return nil
		}
	}
	return fmt.Errorf("No acceptable hash function available; server supports %s but we require %s. Hint: you may need to set the hash function appropriately in the [build] section of your config", fns, systemFn)
}

// digestEnum returns a proto enum for the digest function of given name (as we name them in config)
func (c *Client) digestEnum() pb.DigestFunction_Value {
	switch c.state.Config.Build.HashFunction {
	case "sha256":
		return pb.DigestFunction_SHA256
	case "sha1":
		return pb.DigestFunction_SHA1
	default:
		return pb.DigestFunction_UNKNOWN // Shouldn't get here
	}
}

func (c *Client) SubrepoFS(target *core.BuildTarget, root string) iofs.FS {
	c.outputMutex.RLock()
	defer c.outputMutex.RUnlock()

	tree := c.subrepoTrees[target.Label]
	return remotefs.New(c.remoteFSClient, tree, root)
}

// Build executes a remote build of the given target.
func (c *Client) Build(target *core.BuildTarget) (*core.BuildMetadata, error) {
	if err := c.CheckInitialised(); err != nil {
		return nil, err
	}
	metadata, ar, _, err := c.build(target)
	if err != nil {
		return metadata, err
	}
	if c.state.TargetHasher != nil {
		hash, _ := hex.DecodeString(c.outputHash(ar))
		c.state.TargetHasher.SetHash(target, hash)
	}

	c.setOutputsFromMetadata(target, metadata)

	if c.state.ShouldDownload(target) {
		c.state.LogBuildResult(target, core.TargetBuilding, "Downloading")
		if err := c.Download(target); err != nil {
			return metadata, err
		}
		// TODO(peterebden): Should this not just be part of Download()?
		if err := c.downloadData(target); err != nil {
			return metadata, err
		}
	}
	return metadata, nil
}

// downloadData downloads all the runtime data for a target, recursively.
func (c *Client) downloadData(target *core.BuildTarget) error {
	var g errgroup.Group
	for _, datum := range target.AllData() {
		if l, ok := datum.Label(); ok {
			t := c.state.Graph.TargetOrDie(l)
			g.Go(func() error {
				if err := c.Download(t); err != nil {
					return err
				}
				return c.downloadData(t)
			})
		}
	}
	return g.Wait()
}

// Run runs a target on the remote executors.
func (c *Client) Run(target *core.BuildTarget) error {
	if err := c.CheckInitialised(); err != nil {
		return err
	}
	cmd, digest, err := c.uploadAction(target, false, true, 0)
	if err != nil {
		return err
	}
	// 24 hours is kind of an arbitrarily long timeout. Basically we just don't want to limit it here.
	_, _, err = c.execute(target, cmd, digest, false, false, 0)
	return err
}

// build implements the actual build of a target.
func (c *Client) build(target *core.BuildTarget) (*core.BuildMetadata, *pb.ActionResult, *pb.Digest, error) {
	needStdout := target.PostBuildFunction != nil
	// If we're gonna stamp the target, first check the unstamped equivalent that we store results under.
	// This implements the rules of stamp whereby we don't force rebuilds every time e.g. the SCM revision changes.
	var unstampedDigest *pb.Digest
	if target.Stamp {
		command, digest, err := c.buildAction(target, false, false, 0)
		if err != nil {
			return nil, nil, nil, err
		} else if metadata, ar := c.maybeRetrieveResults(target, command, digest, false, needStdout, 0); metadata != nil {
			c.unstampedBuildActionDigests.Put(target.Label, digest)
			return metadata, ar, digest, nil
		}
		unstampedDigest = digest
	}
	command, stampedDigest, err := c.buildAction(target, false, true, 0)
	if err != nil {
		return nil, nil, nil, err
	}
	metadata, ar, err := c.execute(target, command, stampedDigest, false, needStdout, 0)
	if target.Stamp && err == nil {
		err = c.verifyActionResult(target, command, unstampedDigest, ar, c.state.Config.Remote.VerifyOutputs, false)
		if err == nil {
			// Store results under unstamped digest too.
			c.locallyCacheResults(target, unstampedDigest, metadata)
		}
		c.client.UpdateActionResult(context.Background(), &pb.UpdateActionResultRequest{
			InstanceName: c.instance,
			ActionDigest: unstampedDigest,
			ActionResult: ar,
		})
		c.unstampedBuildActionDigests.Put(target.Label, unstampedDigest)
	} else {
		c.unstampedBuildActionDigests.Put(target.Label, stampedDigest)
	}
	return metadata, ar, stampedDigest, err
}

// Download downloads outputs for the given target.
func (c *Client) Download(target *core.BuildTarget) error {
	if target.Local {
		return nil // No download needed since this target was built locally
	}
	return c.download(target, func() error {
		buildAction := c.unstampedBuildActionDigests.Get(target.Label)
		file := core.AcquireExclusiveFileLock(target.BuildLockFile())
		defer core.ReleaseFileLock(file)

		// This is a bit of a grungy hack to avoid clobbering outputs.
		// See https://github.com/thought-machine/please/issues/2886
		if target.IsFilegroup {
			for _, t := range target.AllSources() {
				if l, ok := t.Label(); ok {
					if l.PackageName == target.Label.PackageName && l.Subrepo == target.Label.Subrepo {
						t := c.state.Graph.TargetOrDie(l)
						file := core.AcquireExclusiveFileLock(t.BuildLockFile())
						defer core.ReleaseFileLock(file)
					}
				}
			}
		}

		if c.outputsExist(target, buildAction) {
			log.Debug("Not downloading outputs for %s, they're already up-to-date", target)
			return nil
		}
		_, ar := c.retrieveResults(target, nil, buildAction, false, false, 0)
		if ar == nil {
			return fmt.Errorf("Failed to retrieve action result for %s", target)
		}
		return c.reallyDownload(target, buildAction, ar)
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
	if err := c.downloadActionOutputs(context.Background(), ar, target); err != nil {
		return c.wrapActionErr(err, digest)
	}
	c.recordAttrs(target, digest)
	log.Debug("Downloaded outputs for %s", target)
	return nil
}

func (c *Client) downloadActionOutputs(ctx context.Context, ar *pb.ActionResult, target *core.BuildTarget) error {
	// Ensure none of the outputs have temp suffixes on them.
	for _, f := range ar.OutputFiles {
		f.Path = target.GetRealOutput(f.Path)
	}
	for _, d := range ar.OutputDirectories {
		d.Path = target.GetRealOutput(d.Path)
	}
	for _, s := range ar.OutputSymlinks {
		s.Path = target.GetRealOutput(s.Path)
	}
	// We can download straight into the out dir if there are no outdirs to worry about
	if len(target.OutputDirectories) == 0 {
		_, err := c.client.DownloadActionOutputs(ctx, ar, target.OutDir(), c.fileMetadataCache)
		return err
	}

	if _, err := c.client.DownloadActionOutputs(ctx, ar, target.TmpDir(), c.fileMetadataCache); err != nil {
		return err
	}

	if err := moveTmpFilesToOutDir(target); err != nil {
		return fmt.Errorf("failed to move downloaded action output from target tmp dir to out dir: %w", err)
	}

	return nil
}

// DownloadInputs downloads all the inputs of a remotely built target into the target directory.
func (c *Client) DownloadInputs(target *core.BuildTarget, targetDir string, isTest bool) error {
	var pbDir *pb.Directory
	// first ensure all inputs are in the CAS
	err := c.uploadBlobs(func(ch chan<- *uploadinfo.Entry) error {
		defer close(ch)

		var err error
		pbDir, err = c.uploadInputs(ch, target, isTest)
		return err
	})
	if err != nil {
		return err
	}

	dirDigest, err := digest.NewFromMessage(pbDir)
	if err != nil {
		return fmt.Errorf("could not calculate digest for directory proto: %w", err)
	}

	if err := fs.RemoveAll(targetDir); err != nil {
		return fmt.Errorf("could not delete target directory %q: %w", targetDir, err)
	}

	if _, _, err = c.client.DownloadDirectory(context.Background(), dirDigest, targetDir, c.fileMetadataCache); err != nil {
		return err
	}

	return nil
}

// moveTmpFilesToOutDir moves files from the target tmp dir to the out dir, handling output directories as well
func moveTmpFilesToOutDir(target *core.BuildTarget) error {
	defer fs.RemoveAll(target.TmpDir())

	// Copy the contents of the output_dirs to the target output dir
	for _, outDir := range target.OutputDirectories {
		dir := filepath.Join(target.TmpDir(), outDir.Dir())
		if err := moveDirToOutDir(target, dir); err != nil {
			return fmt.Errorf("failed to copy output directory %v to target output dir: %v", outDir.Dir(), err)
		}

		// Remove the dir so it doesn't get picked up as an output in the next step
		if err := fs.RemoveAll(dir); err != nil {
			return err
		}
	}

	return moveDirToOutDir(target, target.TmpDir())
}

func moveDirToOutDir(target *core.BuildTarget, dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, f := range files {
		oldPath := filepath.Join(dir, f.Name())
		newPath := filepath.Join(target.OutDir(), f.Name())
		if err := fs.RecursiveCopy(oldPath, newPath, target.OutMode()); err != nil {
			return err
		}
	}
	return nil
}

// Test executes a remote test of the given target.
// It returns the results (and coverage if appropriate) as bytes to be parsed elsewhere.
func (c *Client) Test(target *core.BuildTarget, run int) (metadata *core.BuildMetadata, err error) {
	if err := c.CheckInitialised(); err != nil {
		return nil, err
	}
	command, digest, err := c.buildAction(target, true, false, run)
	if err != nil {
		return nil, err
	}
	metadata, ar, err := c.execute(target, command, digest, true, false, run)

	if ar != nil {
		_, dlErr := c.client.DownloadActionOutputs(context.Background(), ar, target.TestDir(run), c.fileMetadataCache)
		if dlErr != nil {
			log.Warningf("%v: failed to download test outputs: %v", target.Label, dlErr)
		}
	}
	return metadata, err
}

// retrieveResults retrieves target results from where it can (either from the local cache or from remote).
// It returns nil if it cannot be retrieved.
func (c *Client) retrieveResults(target *core.BuildTarget, command *pb.Command, digest *pb.Digest, needStdout, isTest bool, run int) (*core.BuildMetadata, *pb.ActionResult) {
	// First see if this execution is cached locally
	if metadata, ar := c.retrieveLocalResults(target, digest); metadata != nil {
		log.Debug("Got locally cached results for %s %s (age %s)", target.Label, c.actionURL(digest, true), time.Since(metadata.Timestamp).Truncate(time.Second))
		metadata.Cached = true
		return metadata, ar
	}
	c.logActionResult(target, run, "Checking remote...", "")
	// Now see if it is cached on the remote server
	start := time.Now()
	if ar, err := c.client.GetActionResult(context.Background(), &pb.GetActionResultRequest{
		InstanceName: c.instance,
		ActionDigest: digest,
		InlineStdout: needStdout,
	}); err == nil {
		// This action already exists and has been cached.
		remoteCacheReadDuration.WithLabelValues(metrics.CILabel).Observe(float64(time.Since(start).Milliseconds()))
		if metadata, err := c.buildMetadata(target, ar, needStdout, false); err == nil {
			log.Debug("Got remotely cached results for %s %s", target.Label, c.actionURL(digest, true))
			if command != nil {
				err = c.verifyActionResult(target, command, digest, ar, c.state.Config.Remote.VerifyOutputs, isTest)
			}
			if err == nil {
				c.locallyCacheResults(target, digest, metadata)
				metadata.Cached = true
				return metadata, ar
			}
			log.Debug("Remotely cached results for %s were missing some outputs, forcing a rebuild: %s", target.Label, err)
		}
	}
	return nil, nil
}

// maybeRetrieveResults is like retrieveResults but only retrieves if we aren't forcing a rebuild of the target
// (i.e. not if we're doing plz build --rebuild or plz test --rerun).
func (c *Client) maybeRetrieveResults(target *core.BuildTarget, command *pb.Command, digest *pb.Digest, isTest, needStdout bool, run int) (*core.BuildMetadata, *pb.ActionResult) {
	if !c.state.ShouldRebuild(target) && !(c.state.NeedTests && isTest && c.state.ForceRerun) {
		if metadata, ar := c.retrieveResults(target, command, digest, needStdout, isTest, run); metadata != nil {
			return metadata, ar
		}
	}
	return nil, nil
}

// execute submits an action to the remote executor and monitors its progress.
// The returned ActionResult may be nil on failure.
func (c *Client) execute(target *core.BuildTarget, command *pb.Command, digest *pb.Digest, isTest, needStdout bool, run int) (*core.BuildMetadata, *pb.ActionResult, error) {
	if !isTest || (!c.state.ForceRerun && c.state.NumTestRuns == 1) {
		if metadata, ar := c.maybeRetrieveResults(target, command, digest, isTest, needStdout, run); metadata != nil {
			return metadata, ar, nil
		}
	}
	// We didn't actually upload the inputs before, so we must do so now.
	command, digest, err := c.uploadAction(target, isTest, false, run)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to upload build action: %s", err)
	}
	// Remote actions & filegroups get special treatment at this point.
	if target.IsFilegroup {
		// Filegroups get special-cased since they are just a movement of files.
		return c.buildFilegroup(target, command, digest)
	} else if target.IsRemoteFile {
		return c.fetchRemoteFile(target, digest)
	} else if target.IsTextFile {
		return c.buildTextFile(c.state, target, command, digest)
	}

	// We should skip the cache lookup (and override any existing action result) if we --rebuild, or --rerun and this is
	// one fo the targets we're testing or building.
	skipCacheLookup := (isTest && (c.state.ForceRerun || c.state.NumTestRuns != 1)) || (!isTest && c.state.ForceRebuild)
	skipCacheLookup = skipCacheLookup && c.state.IsOriginalTarget(target)

	return c.reallyExecute(target, command, digest, needStdout, isTest, skipCacheLookup, run)
}

// reallyExecute is like execute but after the initial cache check etc.
// The action & sources must have already been uploaded.
func (c *Client) reallyExecute(target *core.BuildTarget, command *pb.Command, digest *pb.Digest, needStdout, isTest, skipCacheLookup bool, run int) (*core.BuildMetadata, *pb.ActionResult, error) {
	executing := false
	c.logActionResult(target, run, "Submitting job...", "")
	updateProgress := func(metadata *pb.ExecuteOperationMetadata) {
		if c.state.Config.Remote.DisplayURL != "" {
			log.Debug("Remote progress for %s: %s%s", target.Label, metadata.Stage, c.actionURL(metadata.ActionDigest, true))
		}
		worker := ""
		if metadata.PartialExecutionMetadata != nil {
			worker = metadata.PartialExecutionMetadata.Worker
		}
		switch metadata.Stage {
		case pb.ExecutionStage_CACHE_CHECK:
			c.logActionResult(target, run, "Checking cache...", worker)
		case pb.ExecutionStage_QUEUED:
			c.logActionResult(target, run, "Queued", worker)
		case pb.ExecutionStage_EXECUTING:
			executing = true
			if target.State() <= core.Built {
				c.logActionResult(target, run, "Building...", worker)
			} else {
				c.logActionResult(target, run, "Testing...", worker)
			}
		case pb.ExecutionStage_COMPLETED:
			c.logActionResult(target, run, "Completed", worker)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		for i := 1; i < 1000000; i++ {
			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Minute):
				description := "queued"
				if executing {
					description = "executing"
				}
				if i == 1 {
					log.Notice("%s still %s after 1 minute", target, description)
				} else {
					log.Notice("%s still %s after %d minutes", target, description, i)
				}
			}
		}
	}()

	resp, err := c.client.ExecuteAndWaitProgress(c.contextWithMetadata(target), &pb.ExecuteRequest{
		InstanceName:    c.instance,
		ActionDigest:    digest,
		SkipCacheLookup: skipCacheLookup,
	}, updateProgress)
	log.Debug("completed ExecuteAndWaitProgress() for %v", target.Label)

	if err != nil {
		// Handle timing issues if we try to resume an execution as it fails. If we get a
		// "not found" we might find that it's already been completed and we can't resume.
		if status.Code(err) == codes.NotFound {
			if metadata, ar := c.retrieveResults(target, command, digest, needStdout, isTest, run); metadata != nil {
				return metadata, ar, nil
			}
		}
		return nil, nil, c.wrapActionErr(fmt.Errorf("Failed to execute %s: %s", target, err), digest)
	}
	switch result := resp.Result.(type) {
	case *longrunningpb.Operation_Error:
		// We shouldn't really get here - the rex API requires servers to always
		// use the response field instead of error.
		return nil, nil, convertError(result.Error)
	case *longrunningpb.Operation_Response:
		response := &pb.ExecuteResponse{}
		if err := result.Response.UnmarshalTo(response); err != nil {
			log.Error("Failed to deserialise execution response: %s", err)
			return nil, nil, err
		}
		if response.CachedResult {
			c.logActionResult(target, run, "Cached", "")
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
						respErr = fmt.Errorf("%w\nAction URL: %s", respErr, url)
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
		metadata, err := c.buildMetadata(target, response.Result, needStdout || failed, failed)
		logResponseTimings(target, response.Result)
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
			return metadata, response.Result, err
		} else if err != nil {
			return nil, nil, err
		}
		log.Debug("Completed remote build action for %s", target)
		if err := c.verifyActionResult(target, command, digest, response.Result, c.state.Config.Remote.VerifyOutputs && !isTest, isTest); err != nil {
			return metadata, response.Result, err
		}
		c.locallyCacheResults(target, digest, metadata)
		return metadata, response.Result, nil
	default:
		if !resp.Done {
			log.Error("Received an incomplete response for %s: %#v", target, resp)
			return nil, nil, fmt.Errorf("Received an incomplete response for %s", target)
		}
		return nil, nil, fmt.Errorf("Unknown response type (was a %T): %#v", resp.Result, resp) // Shouldn't get here
	}
}

func logResponseTimings(target *core.BuildTarget, ar *pb.ActionResult) {
	if ar != nil && ar.ExecutionMetadata != nil {
		startTime := ar.ExecutionMetadata.ExecutionStartTimestamp.AsTime()
		endTime := ar.ExecutionMetadata.ExecutionCompletedTimestamp.AsTime()
		inputFetchStartTime := ar.ExecutionMetadata.InputFetchStartTimestamp.AsTime()
		inputFetchEndTime := ar.ExecutionMetadata.InputFetchCompletedTimestamp.AsTime()
		log.Debug("Completed remote build action for %s; input fetch %s, build time %s", target, inputFetchEndTime.Sub(inputFetchStartTime), endTime.Sub(startTime))
	}
}

// PrintHashes prints the action hashes for a target.
func (c *Client) PrintHashes(target *core.BuildTarget, isTest bool) {
	actionDigest := c.unstampedBuildActionDigests.Get(target.Label)
	fmt.Printf(" Action: %7d bytes: %s\n", actionDigest.SizeBytes, actionDigest.Hash)
	if c.state.Config.Remote.DisplayURL != "" {
		fmt.Printf("    URL: %s\n", c.actionURL(actionDigest, false))
	}
}

// DataRate returns an estimate of the current in/out RPC data rates in bytes per second.
func (c *Client) DataRate() (int, int, int, int) {
	return c.stats.DataRate()
}

// fetchRemoteFile sends a request to fetch a file using the remote asset API.
func (c *Client) fetchRemoteFile(target *core.BuildTarget, actionDigest *pb.Digest) (*core.BuildMetadata, *pb.ActionResult, error) {
	c.state.LogBuildResult(target, core.TargetBuilding, "Downloading...")
	urls := target.AllURLs(c.state)
	req := &fpb.FetchBlobRequest{
		InstanceName: c.instance,
		Timeout:      durationpb.New(target.BuildTimeout),
		Uris:         urls,
	}
	if c.state.VerifyHashes && (!c.state.NeedHashesOnly || !c.state.IsOriginalTargetOrParent(target)) {
		if sri := subresourceIntegrity(target); sri != "" {
			req.Qualifiers = []*fpb.Qualifier{{
				Name:  "checksum.sri",
				Value: sri,
			}}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), target.BuildTimeout)
	defer cancel()
	resp, err := c.fetchClient.FetchBlob(ctx, req)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to download file: %s", err)
	}
	c.state.LogBuildResult(target, core.TargetBuilding, "Downloaded.")
	// If we get here, the blob exists in the CAS. Create an ActionResult corresponding to it.
	outs := target.Outputs()
	ar := &pb.ActionResult{
		OutputFiles: []*pb.OutputFile{{
			Path:         outs[0],
			Digest:       resp.BlobDigest,
			IsExecutable: target.IsBinary,
		}},
	}
	if _, err := c.client.UpdateActionResult(context.Background(), &pb.UpdateActionResultRequest{
		InstanceName: c.instance,
		ActionDigest: actionDigest,
		ActionResult: ar,
	}); err != nil {
		return nil, nil, fmt.Errorf("Error updating action result: %s", err)
	}
	md, err := c.buildMetadata(target, ar, false, false)
	return md, ar, err
}

// buildFilegroup "builds" a single filegroup target.
func (c *Client) buildFilegroup(target *core.BuildTarget, command *pb.Command, actionDigest *pb.Digest) (*core.BuildMetadata, *pb.ActionResult, error) {
	inputDir, err := c.uploadInputDir(nil, target, false) // We don't need to actually upload the inputs here, that is already done.
	if err != nil {
		return nil, nil, err
	}
	ar := &pb.ActionResult{}
	if err := c.uploadBlobs(func(ch chan<- *uploadinfo.Entry) error {
		defer close(ch)
		inputDir.Build(ch)
		for _, out := range command.OutputPaths {
			if d, f := inputDir.Node(filepath.Join(target.PackageDir(), out)); d != nil {
				entry, digest := c.protoEntry(inputDir.Tree(filepath.Join(target.PackageDir(), out)))
				ch <- entry
				ar.OutputDirectories = append(ar.OutputDirectories, &pb.OutputDirectory{
					Path:       out,
					TreeDigest: digest,
				})
			} else if f != nil {
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
	if _, err := c.client.UpdateActionResult(context.Background(), &pb.UpdateActionResultRequest{
		InstanceName: c.instance,
		ActionDigest: actionDigest,
		ActionResult: ar,
	}); err != nil {
		return nil, nil, fmt.Errorf("Error updating action result: %s", err)
	}
	md, err := c.buildMetadata(target, ar, false, false)
	if err != nil {
		return nil, nil, err
	}
	c.locallyCacheResults(target, actionDigest, md)
	return md, ar, nil
}

// buildTextFile "builds" uploads a text file to the CAS
func (c *Client) buildTextFile(state *core.BuildState, target *core.BuildTarget, command *pb.Command, actionDigest *pb.Digest) (*core.BuildMetadata, *pb.ActionResult, error) {
	ar := &pb.ActionResult{}
	if err := c.uploadBlobs(func(ch chan<- *uploadinfo.Entry) error {
		defer close(ch)
		if len(command.OutputPaths) != 1 {
			return fmt.Errorf("text_file %s should have a single output, has %d", target.Label, len(command.OutputPaths))
		}
		content, err := target.GetFileContent(state)
		if err != nil {
			return err
		}
		entry := uploadinfo.EntryFromBlob([]byte(content))
		ch <- entry
		ar.OutputFiles = append(ar.OutputFiles, &pb.OutputFile{
			Path:         command.OutputPaths[0],
			Digest:       entry.Digest.ToProto(),
			IsExecutable: target.IsBinary,
		})
		return nil
	}); err != nil {
		return nil, nil, err
	}
	if _, err := c.client.UpdateActionResult(context.Background(), &pb.UpdateActionResultRequest{
		InstanceName: c.instance,
		ActionDigest: actionDigest,
		ActionResult: ar,
	}); err != nil {
		return nil, nil, fmt.Errorf("Error updating action result: %s", err)
	}
	md, err := c.buildMetadata(target, ar, false, false)
	if err != nil {
		return nil, nil, err
	}
	c.locallyCacheResults(target, actionDigest, md)
	return md, ar, nil
}

// logActionResult logs the state of an action while it's building or testing
func (c *Client) logActionResult(target *core.BuildTarget, run int, message, worker string) {
	if worker != "" {
		message += " (on " + worker + ")"
	}
	if target.State() <= core.Built {
		c.state.LogBuildResult(target, core.TargetBuilding, message)
	} else {
		c.state.LogTestRunning(target, run, core.TargetTesting, message)
	}
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
