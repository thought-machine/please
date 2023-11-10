// Package fs provides an io/fs.FS implementation over the remote API
package fs

import (
	"context"
	"errors"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/client"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/digest"
	pb "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"google.golang.org/protobuf/proto"
	"io"
	iofs "io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Client interface {
	ReadBlob(ctx context.Context, d digest.Digest) ([]byte, *client.MovedBytesMetadata, error)
}

// fs is an io/fs.FS implemented on top of a REAPI directory. This will download files on demand, as they are needed.
type fs struct {
	c   Client
	dir *pb.Directory
}

// New creates a new filesystem on top of the given proto, using client to download files on demand.
func New(c Client, root *pb.Directory) iofs.FS {
	return &fs{
		c:   c,
		dir: root,
	}
}

// Open opens the file with the given name
func (fs *fs) Open(name string) (iofs.File, error) {
	if len(fs.dir.Symlinks) > 0 {
		panic("not implemented yet")
	}
	for _, f := range fs.dir.Files {
		if f.Name == name {
			d, err := digest.NewFromProto(f.Digest)
			if err != nil {
				return nil, err
			}
			bs, _, err := fs.c.ReadBlob(context.Background(), d)
			if err != nil {
				return nil, err
			}
			return &file{
				bs: bs,
				info: info{
					isDir:   false,
					size:    int64(len(bs)),
					mode:    iofs.FileMode(f.NodeProperties.UnixMode.Value),
					modTime: f.NodeProperties.GetMtime().AsTime(),
					name:    f.Name,
				},
			}, nil
		}
	}
	name, rest, _ := strings.Cut(name, string(filepath.Separator))
	for _, d := range fs.dir.Directories {
		if d.Name == name {
			dg, err := digest.NewFromProto(d.Digest)
			if err != nil {
				return nil, err
			}
			bs, _, err := fs.c.ReadBlob(context.Background(), dg)
			if err != nil {
				return nil, err
			}
			dirPb := &pb.Directory{}
			if err := proto.Unmarshal(bs, dirPb); err != nil {
				return nil, err
			}
			if rest == "" {
				return &dir{
					info: info{
						mode:    iofs.FileMode(dirPb.NodeProperties.UnixMode.Value),
						modTime: dirPb.NodeProperties.GetMtime().AsTime(),
						name:    name,
						isDir:   true,
					},
					pb: dirPb,
				}, nil
			}
			return New(fs.c, dirPb).Open(rest)
		}
	}
	return nil, iofs.ErrNotExist
}

type file struct {
	bs []byte
	info
}

func (b *file) Stat() (iofs.FileInfo, error) {
	return b, nil
}

func (b *file) Read(bytes []byte) (int, error) {
	for i := range bytes {
		if i == len(b.bs) {
			return i, io.EOF
		}
		if i == len(bytes) {
			return i, nil
		}
		bytes[i] = b.bs[i]
	}
	return len(b.bs), nil
}

func (b *file) Close() error {
	return nil
}

type dir struct {
	pb *pb.Directory
	info
}

// ReadDir is a slightly incorrect implementation of ReadDir. It doesn't return all the information you would normally
// have with a filesystem. We can't know this information without downloading more digests from the client, however we
// likely will never need this. This is enough to facilitate globbing.
func (p *dir) ReadDir(n int) ([]iofs.DirEntry, error) {
	if len(p.pb.Symlinks) > 0 {
		panic("not implemented yet")
	}
	dirSize := n
	if n <= 0 {
		dirSize = len(p.pb.Files) + len(p.pb.Symlinks) + len(p.pb.Files)
	}
	ret := make([]iofs.DirEntry, 0, dirSize)
	for _, dir := range p.pb.Directories {
		if n > 0 && len(ret) == n {
			return ret, nil
		}
		ret = append(ret, &info{
			name:     dir.Name,
			isDir:    true,
			typeMode: os.ModeDir,
			mode:     0, // We can't know this without downloading the Directory proto
		})
	}
	for _, file := range p.pb.Files {
		if n > 0 && len(ret) == n {
			return ret, nil
		}
		ret = append(ret, &info{
			name: file.Name,
			mode: os.FileMode(file.NodeProperties.UnixMode.Value),
			size: 0, // We can't know this without downloading the file.
		})
	}
	return ret, nil
}

func (p *dir) Stat() (iofs.FileInfo, error) {
	return p, nil
}

func (p *dir) Read(_ []byte) (int, error) {
	return 0, errors.New("attempt to read a directory")
}

func (p *dir) Close() error {
	return nil
}
