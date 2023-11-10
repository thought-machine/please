// Package fs provides an io/fs.FS implementation over the remote API
package fs

import (
	"context"
	"errors"
	"io"
	iofs "io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bazelbuild/remote-apis-sdks/go/pkg/digest"
	pb "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"google.golang.org/protobuf/proto"
)

type Client interface {
	ReadBlob(ctx context.Context, d digest.Digest) ([]byte, error)
}

type fs struct {
	c   Client
	dir *pb.Directory
}

func New(c Client, root *pb.Directory) iofs.FS {
	return &fs{
		c:   c,
		dir: root,
	}
}

func (f *fs) Open(name string) (iofs.File, error) {
	if len(f.dir.Symlinks) > 0 {
		panic("not implemented yet")
	}
	for _, file := range f.dir.Files {
		if file.Name == name {
			d, err := digest.NewFromProto(file.Digest)
			if err != nil {
				return nil, err
			}
			bs, err := f.c.ReadBlob(context.Background(), d)
			if err != nil {
				return nil, err
			}
			return &byteFile{
				bs: bs,
				info: info{
					isDir: false,
					size:  int64(len(bs)),
					mode:  iofs.FileMode(file.NodeProperties.UnixMode.Value),
					name:  file.Name,
				},
			}, nil
		}
	}
	name, rest, _ := strings.Cut(name, string(filepath.Separator))
	for _, dir := range f.dir.Directories {
		if dir.Name == name {
			d, err := digest.NewFromProto(dir.Digest)
			if err != nil {
				return nil, err
			}
			dir, err := f.c.ReadBlob(context.Background(), d)
			if err != nil {
				return nil, err
			}
			dirPb := &pb.Directory{}
			if err := proto.Unmarshal(dir, dirPb); err != nil {
				return nil, err
			}
			if rest == "" {
				return &pbDir{
					info: info{
						mode:  iofs.FileMode(dirPb.NodeProperties.UnixMode.Value),
						name:  name,
						isDir: true,
					},
					pb: dirPb,
				}, nil
			}
			return New(f.c, dirPb).Open(rest)
		}
	}
	return nil, iofs.ErrNotExist
}

type info struct {
	name     string
	isDir    bool
	size     int64
	mode     os.FileMode
	typeMode os.FileMode
}

func (i *info) Type() iofs.FileMode {
	return i.typeMode
}

func (i *info) Info() (iofs.FileInfo, error) {
	return i, nil
}

func (i *info) Name() string {
	return i.name
}

func (i *info) Size() int64 {
	return i.size
}

func (i *info) Mode() iofs.FileMode {
	return i.mode
}

func (i *info) ModTime() time.Time {
	return time.Now()
}

func (i *info) IsDir() bool {
	return false
}

func (i *info) Sys() any {
	return nil
}

type byteFile struct {
	bs []byte
	info
}

func (b *byteFile) Stat() (iofs.FileInfo, error) {
	return b, nil
}

func (b *byteFile) Read(bytes []byte) (int, error) {
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

func (b *byteFile) Close() error {
	return nil
}

type pbDir struct {
	pb *pb.Directory
	info
}

// ReadDir is a slightly incorrect implementation of ReadDir. It doesn't return all the information you would normally
// have with a filesystem. We can't know this information without downloading more digests from the client, however we
// likely will never need this. This is enough to facilitate globbing.
func (p *pbDir) ReadDir(n int) ([]iofs.DirEntry, error) {
	if len(p.pb.Symlinks) > 0 {
		panic("not implemented yet")
	}
	dirSize := n
	if n <= 0 {
		dirSize = len(p.pb.Files) + len(p.pb.Symlinks) + len(p.pb.Files)
	}
	ret := make([]iofs.DirEntry, dirSize)
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

func (p *pbDir) Stat() (iofs.FileInfo, error) {
	return p, nil
}

func (p *pbDir) Read(_ []byte) (int, error) {
	return 0, errors.New("attempt to read a directory")
}

func (p *pbDir) Close() error {
	return nil
}
