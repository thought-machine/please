package fs

import (
	"context"
	iofs "io/fs"
	"os"
	"testing"

	"github.com/bazelbuild/remote-apis-sdks/go/pkg/client"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/digest"
	pb "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	// Directory structure:
	// . (root)
	// |- foo (file containing wibble wibble wibble)
	// |- bar
	//    |- empty (an empty directory)
	//    |- foo (same file as above)
	//    |- example.go
	//    |- example_test.go
	//    |- link (a symlink to ../foo i.e. foo in the root dir)
	//    |- badlink (a symlink to ../../foo which is root/.. i.e. invalid)

	fooDigest := newDigest("foo")

	foo := &pb.FileNode{
		Name: "foo",
		NodeProperties: &pb.NodeProperties{UnixMode: &wrappers.UInt32Value{
			Value: 0777,
		}},
		Digest: fooDigest.ToProto(),
	}

	empty := &pb.Directory{
		NodeProperties: &pb.NodeProperties{UnixMode: &wrappers.UInt32Value{
			Value: 0777,
		}}}
	emptyDigest, err := digest.NewFromMessage(empty)
	require.NoError(t, err)

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
			{
				Name:   "badlink",
				Target: "../../foo",
				NodeProperties: &pb.NodeProperties{UnixMode: &wrappers.UInt32Value{
					Value: 0777,
				}},
			},
		},
		Directories: []*pb.DirectoryNode{
			{
				Name:   "empty",
				Digest: emptyDigest.ToProto(),
			},
		},
		NodeProperties: &pb.NodeProperties{UnixMode: &wrappers.UInt32Value{
			Value: 0777,
		}},
	}

	barDigest, err := digest.NewFromMessage(bar)
	require.NoError(t, err)

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

	fc := &fakeClient{
		results: map[digest.Digest][]byte{
			fooDigest: []byte("wibble wibble wibble"),
		},
	}
	tree := &pb.Tree{
		Root: root,
		Children: []*pb.Directory{
			bar,
			empty,
		},
	}

	fs := New(fc, tree)
	bs, err := iofs.ReadFile(fs, "foo")
	require.NoError(t, err)
	assert.Equal(t, "wibble wibble wibble", string(bs))

	bs, err = iofs.ReadFile(fs, "bar/foo")
	require.NoError(t, err)
	assert.Equal(t, "wibble wibble wibble", string(bs))

	bs, err = iofs.ReadFile(fs, "bar/link")
	require.NoError(t, err)
	assert.Equal(t, "wibble wibble wibble", string(bs))

	bs, err = iofs.ReadFile(fs, "bar/badlink")
	require.Error(t, err)
	assert.ErrorIs(t, err, os.ErrNotExist)

	entries, err := iofs.ReadDir(fs, "bar")
	require.NoError(t, err)
	assert.Len(t, entries, 6)

	for _, e := range entries {
		i, err := e.Info()
		require.NoError(t, err)
		// We set them all to 0777 above
		assert.Equal(t, iofs.FileMode(0777), i.Mode(), "%v mode was wrong", e.Name())
	}

	matches, err := iofs.Glob(fs, "bar/*.go")
	require.NoError(t, err)
	assert.Len(t, matches, 2)
	assert.ElementsMatch(t, matches, []string{"bar/example.go", "bar/example_test.go"})
}
