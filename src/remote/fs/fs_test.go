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

var fooContent = "wibble wibble wibble"

type fakeClient struct {
	results map[digest.Digest][]byte
}

func (f *fakeClient) ReadBlob(_ context.Context, d digest.Digest) ([]byte, *client.MovedBytesMetadata, error) {
	res := f.results[d]
	return res, nil, nil
}

func newDigest(str string) digest.Digest {
	return digest.NewFromBlob([]byte(str))
}

// getTree returns a pb.Tree proto representing the following dir structure:
// . (root)
// |- foo (file containing wibble wibble wibble)
// |- bar
//
//	|- empty (an empty directory)
//	|- foo (same file as above)
//	|- example.go (not in CAS)
//	|- example_test.go (not in CAS)
//	|- link (a symlink to ../foo i.e. foo in the root dir)
//	|- badlink (a symlink to ../../foo which is root/.. i.e. invalid)
func getTree(t *testing.T) (*fakeClient, *pb.Tree) {
	t.Helper()
	fooDigest := newDigest(fooContent)

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
		}},
	}
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
			fooDigest: []byte(fooContent),
		},
	}
	tree := &pb.Tree{
		Root: root,
		Children: []*pb.Directory{
			bar,
			empty,
		},
	}
	return fc, tree
}

func TestReadDir(t *testing.T) {
	fc, tree := getTree(t)
	fs := New(fc, tree, "")

	entries, err := iofs.ReadDir(fs, "bar")
	require.NoError(t, err)
	assert.Len(t, entries, 6)

	for _, e := range entries {
		i, err := e.Info()
		require.NoError(t, err)
		// We set them all to 0777 above
		assert.Equal(t, iofs.FileMode(0777), i.Mode().Perm(), "%v mode was wrong", e.Name())
		if e.Name() == "foo" {
			assert.Equal(t, len([]byte(fooContent)), int(i.Size()))
		}
	}

	entries, err = iofs.ReadDir(fs, ".")
	require.NoError(t, err)
	require.Len(t, entries, 2)
}

func TestGlob(t *testing.T) {
	fc, tree := getTree(t)
	fs := New(fc, tree, "")

	matches, err := iofs.Glob(fs, "bar/*.go")
	require.NoError(t, err)
	assert.Len(t, matches, 2)
	assert.ElementsMatch(t, matches, []string{"bar/example.go", "bar/example_test.go"})
}

func TestReadFile(t *testing.T) {
	fc, tree := getTree(t)

	tests := []struct {
		name           string
		wd             string
		file           string
		expectError    bool
		expectedOutput string
	}{
		{
			name:           "Open file in root",
			wd:             ".",
			file:           "foo",
			expectedOutput: "wibble wibble wibble",
		},
		{
			name:           "Open file in root with .",
			wd:             ".",
			file:           "./foo",
			expectedOutput: "wibble wibble wibble",
		},
		{
			name:           "Open file in dir",
			wd:             ".",
			file:           "bar/foo",
			expectedOutput: "wibble wibble wibble",
		},
		{
			name:           "Open file in dir with .",
			wd:             ".",
			file:           "bar/./foo",
			expectedOutput: "wibble wibble wibble",
		},
		{
			name:           "Open file with working dir",
			wd:             "bar",
			file:           "foo",
			expectedOutput: "wibble wibble wibble",
		},
		{
			name:           "Open symlink",
			wd:             ".",
			file:           "bar/link",
			expectedOutput: "wibble wibble wibble",
		},
		{
			name:           "Open symlink from working dir",
			wd:             "bar",
			file:           "link",
			expectedOutput: "wibble wibble wibble",
		},
		{
			name:        "Open bad symlink",
			wd:          ".",
			file:        "bar/badlink",
			expectError: true,
		},
		{
			name:        "Open missing file",
			wd:          ".",
			file:        "bar/faff",
			expectError: true,
		},
		{
			name:        "Open directory passed file",
			wd:          ".",
			file:        "foo/bar",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bs, err := iofs.ReadFile(New(fc, tree, tc.wd), tc.file)
			if tc.expectError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expectedOutput, string(bs))
		})
	}
}

func TestStat(t *testing.T) {
	fc, tree := getTree(t)

	tests := []struct {
		name           string
		file           string
		expectNotExist bool
		expectedType   os.FileMode
	}{
		{
			name:         "Stat file",
			file:         "bar/example.go",
			expectedType: 0,
		},
		{
			name:         "Stat dir",
			file:         "bar",
			expectedType: os.ModeDir,
		},
		{
			name:         "Stat symlink",
			file:         "bar/link",
			expectedType: os.ModeSymlink,
		},
		{
			name:           "Stat not exist",
			file:           "bar/not_exist.go",
			expectNotExist: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			i, err := iofs.Stat(New(fc, tree, "."), tc.file)
			if tc.expectNotExist {
				assert.True(t, os.IsNotExist(err))
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expectedType.IsDir(), i.Mode().IsDir())
			assert.Equal(t, tc.expectedType.IsRegular(), i.Mode().IsRegular())
			assert.Equal(t, tc.expectedType&os.ModeSymlink != 0, i.Mode()&os.ModeSymlink != 0)
		})
	}
}
