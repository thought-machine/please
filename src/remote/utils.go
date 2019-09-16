package remote

import (
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	pb "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/bazelbuild/remote-apis/build/bazel/semver"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	rpcstatus "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/thought-machine/please/src/core"
)

// sum calculates a checksum for a byte slice.
func (c *Client) sum(b []byte) []byte {
	h := c.state.PathHasher.NewHash()
	h.Write(b)
	return h.Sum(nil)
}

// targetOutputs returns the outputs for a previously executed target.
// If it has not been executed this returns nil.
func (c *Client) targetOutputs(label core.BuildLabel) *pb.Directory {
	c.outputMutex.RLock()
	defer c.outputMutex.RUnlock()
	return c.outputs[label]
}

// setOutputs sets the outputs for a previously executed target.
func (c *Client) setOutputs(label core.BuildLabel, ar *pb.ActionResult) error {
	o := &pb.Directory{
		Files:       make([]*pb.FileNode, len(ar.OutputFiles)),
		Directories: make([]*pb.DirectoryNode, len(ar.OutputDirectories)),
		Symlinks:    make([]*pb.SymlinkNode, len(ar.OutputFileSymlinks)+len(ar.OutputDirectorySymlinks)),
	}
	for i, f := range ar.OutputFiles {
		o.Files[i] = &pb.FileNode{
			Name:         f.Path,
			Digest:       f.Digest,
			IsExecutable: f.IsExecutable,
		}
	}
	for i, d := range ar.OutputDirectories {
		// Awkwardly these are encoded as Trees rather than as anything directly useful.
		// We need a DirectoryNode to feed in as an input later on, but the OutputDirectory
		// we get back is quite a different structure at the top level.
		// TODO(peterebden): Test this for real, this is theoretically OK but will only
		//                   actually work if the remote end has uploaded the relevant
		//                   blobs. If it has not we will probably have to do that here?
		tree := &pb.Tree{}
		if err := c.readByteStreamToProto(d.TreeDigest, tree); err != nil {
			return err
		}
		o.Directories[i] = &pb.DirectoryNode{
			Name:   d.Path,
			Digest: c.digestMessage(tree.Root),
		}
	}
	for i, s := range append(ar.OutputFileSymlinks, ar.OutputDirectorySymlinks...) {
		o.Symlinks[i] = &pb.SymlinkNode{
			Name:   s.Path,
			Target: s.Target,
		}
	}
	c.outputMutex.Lock()
	defer c.outputMutex.Unlock()
	c.outputs[label] = o
	return nil
}

// digestMessage calculates the digest of a proto message as described in the
// Digest message's comments.
func (c *Client) digestMessage(msg proto.Message) *pb.Digest {
	digest, _ := c.digestMessageContents(msg)
	return digest
}

// digestMessageContents is like DigestMessage but returns the serialised contents as well.
func (c *Client) digestMessageContents(msg proto.Message) (*pb.Digest, []byte) {
	b := mustMarshal(msg)
	sum := c.sum(b)
	return &pb.Digest{
		Hash:      hex.EncodeToString(sum[:]),
		SizeBytes: int64(len(b)),
	}, b
}

// mustMarshal encodes a message to a binary string.
func mustMarshal(msg proto.Message) []byte {
	b, err := proto.Marshal(msg)
	if err != nil {
		// Not really sure if there is a valid possibility to bring us here (given that
		// the messages in question have no required fields) so assume it won't happen :)
		log.Fatalf("Failed to marshal message: %s", err)
	}
	return b
}

// lessThan returns true if the given semver instance is less than another one.
func lessThan(a, b *semver.SemVer) bool {
	if a.Major < b.Major {
		return true
	} else if a.Major > b.Major {
		return false
	} else if a.Minor < b.Minor {
		return true
	} else if a.Minor > b.Minor {
		return false
	} else if a.Patch < b.Patch {
		return true
	} else if a.Patch > b.Patch {
		return false
	}
	return a.Prerelease < b.Prerelease
}

// printVer pretty-prints a semver message.
// The default stringing of them is so bad as to be completely unreadable.
func printVer(v *semver.SemVer) string {
	msg := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Prerelease != "" {
		msg += "-" + v.Prerelease
	}
	return msg
}

// toTimestamp converts a time.Time into a protobuf timestamp
func toTimestamp(t time.Time) *timestamp.Timestamp {
	return &timestamp.Timestamp{
		Seconds: t.Unix(),
		Nanos:   int32(t.Nanosecond()),
	}
}

// toTime converts a protobuf timestamp into a time.Time.
// It's like the ptypes one but we ignore errors (we don't generally care that much)
func toTime(ts *timestamp.Timestamp) time.Time {
	t, _ := ptypes.Timestamp(ts)
	return t
}

// extraPerms returns any additional permission bits we should apply for this file.
func extraPerms(file *pb.OutputFile) os.FileMode {
	if file.IsExecutable {
		return 0111
	}
	return 0
}

// extraFilePerms returns any additional permission bits we should apply for this file.
func extraFilePerms(file *pb.FileNode) os.FileMode {
	if file.IsExecutable {
		return 0111
	}
	return 0
}

// IsNotFound returns true if a given error is a "not found" error (which may be treated
// differently, for example if trying to retrieve artifacts that may not be there).
func IsNotFound(err error) bool {
	return status.Code(err) == codes.NotFound
}

// hasChild returns true if a Directory has a child directory by the given name.
func hasChild(dir *pb.Directory, child string) bool {
	for _, d := range dir.Directories {
		if d.Name == child {
			return true
		}
	}
	return false
}

// exhaustChannel reads and discards all messages on the given channel.
func exhaustChannel(ch <-chan *blob) {
	for range ch {
	}
}

// convertError converts a single google.rpc.Status message into a Go error
func convertError(err *rpcstatus.Status) error {
	if err.Code == int32(codes.OK) {
		return nil
	}
	return fmt.Errorf("%s", err.Message)
}

// timeout returns either a build or test timeout from a target.
func timeout(target *core.BuildTarget, test bool) time.Duration {
	if test {
		return target.TestTimeout
	}
	return target.BuildTimeout
}

// outputs returns the outputs of a target, split arbitrarily and inaccurately
// into files and directories.
// After some discussion we are hoping that servers are permissive about this if
// we get it wrong; we prefer to make an effort though as a minor nicety.
func outputs(target *core.BuildTarget) (files, dirs []string) {
	outs := target.Outputs()
	files = make([]string, 0, len(outs))
	for _, out := range outs {
		if !strings.ContainsRune(path.Base(out), '.') && !strings.HasSuffix(out, "file") {
			dirs = append(dirs, out)
		} else {
			files = append(files, out)
		}
	}
	return files, dirs
}
