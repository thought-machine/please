// Package remote provides our interface to the Google remote execution APIs
// (https://github.com/bazelbuild/remote-apis) which Please can use to distribute
// work to remote servers.
package remote

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"time"

	pb "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/bazelbuild/remote-apis/build/bazel/semver"
	"github.com/golang/protobuf/ptypes"
	"github.com/grpc-ecosystem/go-grpc-middleware/retry"
	bs "google.golang.org/genproto/googleapis/bytestream"
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
	initOnce          sync.Once
	state             *core.BuildState
	err               error // for initialisation

	// This is for servers have have multiple instances. Right now we never set it but
	// we keep this here to remind us where it would need to go in the API.
	instance string

	// Server-sent cache properties
	maxBlobBatchSize int64
	cacheWritable    bool

	// Cache this for later
	bashPath string
}

// instance is the singleton client instance for Get()
var instance *Client

// New returns a new Client instance.
// It begins the process of contacting the remote server but does not wait for it.
func New(state *core.BuildState) *Client {
	c := &Client{state: state}
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
		} else if lessThan(&apiVersion, resp.LowApiVersion) || lessThan(resp.HighApiVersion, &apiVersion) {
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
		c.actionCacheClient = pb.NewActionCacheClient(conn)
		c.storageClient = pb.NewContentAddressableStorageClient(conn)
		c.bsClient = bs.NewByteStreamClient(conn)
		// Look this up just once now.
		bash, err := core.LookBuildPath("bash", c.state.Config)
		c.bashPath = bash
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

func (c *Client) GetArtifact() {
}

// Store stores a set of artifacts for a single build target.
func (c *Client) Store(target *core.BuildTarget, key []byte, files []string) error {
	// v0.1: just do BatchUpdateBlobs  <-- we are here
	// v0.2: honour the max size to do ByteStreams
	// v0.3: get the action cache involved
	ar := &pb.ActionResult{
		// We never cache any failed actions so ExitCode is implicitly 0.
		ExecutionMetadata: &pb.ExecutedActionMetadata{
			Worker: c.state.Config.Remote.Name,
			// TODO(peterebden): Add some kind of temporary metadata so we can know at least
			//                   the start/completed timestamps and stdout/stderr.
			//                   We will need stdout at least for post-build functions.
			OutputUploadStartTimestamp: toTimestamp(time.Now()),
		},
	}
	if err := c.uploadBlobs(func(ch chan<- *blob) error {
		defer close(ch)
		for _, file := range files {
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
		}
		return nil
	}); err != nil {
		return err
	}
	// OK, now the blobs are uploaded, we also need to upload the Action itself.
	digest, err := c.uploadAction(target, key)
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
func (c *Client) Retrieve(target *core.BuildTarget, key []byte) error {
	inputRoot, err := c.buildInputRoot(target, false)
	if err != nil {
		return err
	}
	digest := digestMessage(&pb.Action{
		CommandDigest:   digestMessage(c.buildCommand(target, key)),
		InputRootDigest: digestMessage(inputRoot),
		Timeout:         ptypes.DurationProto(target.BuildTimeout),
	})
	ctx, cancel := context.WithTimeout(context.Background(), reqTimeout)
	defer cancel()
	resp, err := c.actionCacheClient.GetActionResult(ctx, &pb.GetActionResultRequest{
		InstanceName: c.instance,
		ActionDigest: digest,
		InlineStdout: target.PostBuildFunction != nil, // We only care in this case.
	})
	if err != nil {
		return err
	}
	mode := target.OutMode()
	return c.downloadBlobs(func(ch chan<- *blob) error {
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
	})
}

// digestDir calculates the digest for a directory.
// It returns Directory protos for the directory and all its (recursive) children.
func (c *Client) digestDir(dir string, children []*pb.Directory) (*pb.Directory, []*pb.Directory, error) {
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}
	d := &pb.Directory{}
	err = c.uploadBlobs(func(ch chan<- *blob) error {
		for _, entry := range entries {
			name := entry.Name()
			fullname := path.Join(dir, name)
			if mode := entry.Mode(); mode&os.ModeDir != 0 {
				dir, descendants, err := c.digestDir(fullname, children)
				if err != nil {
					return err
				}
				d.Directories = append(d.Directories, &pb.DirectoryNode{
					Name:   name,
					Digest: digestMessage(dir),
				})
				children = append(children, descendants...)
				continue
			} else if mode&os.ModeSymlink != 0 {
				target, err := os.Readlink(fullname)
				if err != nil {
					return err
				}
				d.Symlinks = append(d.Symlinks, &pb.SymlinkNode{
					Name:   name,
					Target: target,
				})
				continue
			}
			ch <- &blob{
				File:   fullname,
				Digest: &pb.Digest{SizeBytes: entry.Size()},
			}
		}
		return nil
	})
	return d, children, err
}
