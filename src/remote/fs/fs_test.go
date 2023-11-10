package fs

import (
	"context"
	iofs "io/fs"
	"testing"

	"github.com/bazelbuild/remote-apis-sdks/go/pkg/client"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/digest"
	pb "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

type fakeClient struct {
	results map[digest.Digest][]byte
}

func (f *fakeClient) ReadBlob(ctx context.Context, d digest.Digest) ([]byte, *client.MovedBytesMetadata, error) {
	res := f.results[d]
	return res, nil, nil
}

func newDigest(str string) digest.Digest {
	return digest.NewFromBlob([]byte(str))
}

func TestFS(t *testing.T) {
	fooDigest := newDigest("foo")
	barDigest := newDigest("bar")

	foo := &pb.FileNode{
		Name: "foo",
		NodeProperties: &pb.NodeProperties{UnixMode: &wrappers.UInt32Value{
			Value: 0777,
		}},
		Digest: fooDigest.ToProto(),
	}

	root := &pb.Directory{
		Files: []*pb.FileNode{
			foo,
		},
		Directories: []*pb.DirectoryNode{
			{
				Name:   "bar",
				Digest: barDigest.ToProto(),
			},
		},
	}

	bar := &pb.Directory{
		Files: []*pb.FileNode{
			foo,
			{
				Name:   "example.go",
				Digest: newDigest("example.go").ToProto(),
				NodeProperties: &pb.NodeProperties{UnixMode: &wrappers.UInt32Value{
					Value: 0777,
				}},
			},
			{
				Name:   "example_test.go",
				Digest: newDigest("example_test.go").ToProto(),
				NodeProperties: &pb.NodeProperties{UnixMode: &wrappers.UInt32Value{
					Value: 0777,
				}},
			},
		},
		Symlinks: []*pb.SymlinkNode{
			{
				Name:   "link",
				Target: "../foo",
				NodeProperties: &pb.NodeProperties{UnixMode: &wrappers.UInt32Value{
					Value: 0777,
				}},
			},
		},
		NodeProperties: &pb.NodeProperties{UnixMode: &wrappers.UInt32Value{
			Value: 0777,
		}},
	}

	barBs, err := proto.Marshal(bar)
	require.NoError(t, err)

	fc := &fakeClient{
		results: map[digest.Digest][]byte{
			fooDigest: []byte("wibble wibble wibble"),
			barDigest: barBs,
		},
	}

	fs := New(fc, root)
	bs, err := iofs.ReadFile(fs, "foo")
	require.NoError(t, err)
	assert.Equal(t, "wibble wibble wibble", string(bs))

	bs, err = iofs.ReadFile(fs, "bar/foo")
	require.NoError(t, err)
	assert.Equal(t, "wibble wibble wibble", string(bs))

	bs, err = iofs.ReadFile(fs, "bar/link")
	require.NoError(t, err)
	assert.Equal(t, "wibble wibble wibble", string(bs))

	entries, err := iofs.ReadDir(fs, "bar")
	require.NoError(t, err)
	assert.Len(t, entries, 4)

	matches, err := iofs.Glob(fs, "bar/*.go")
	require.NoError(t, err)
	assert.Len(t, matches, 2)
	assert.ElementsMatch(t, matches, []string{"bar/example.go", "bar/example_test.go"})
}
