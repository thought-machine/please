// Package fs provides an io/fs.FS implementation over the remote API
package fs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	iofs "io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bazelbuild/remote-apis-sdks/go/pkg/client"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/digest"
	pb "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
)

type Client interface {
	ReadBlob(ctx context.Context, d digest.Digest) ([]byte, *client.MovedBytesMetadata, error)
}

// fs is an io/fs.FS implemented on top of a REAPI directory. This will download files as they are needed.
type fs struct {
	c           Client
	root        *pb.Directory
	directories map[digest.Digest]*pb.Directory
	workingDir  string
}

// New creates a new filesystem on top of the given proto, using client to download files on demand.
func New(c Client, tree *pb.Tree, workingDir string) iofs.FS {
	directories := make(map[digest.Digest]*pb.Directory, len(tree.Children))
	for _, child := range tree.Children {
		dg, err := digest.NewFromMessage(child)
		if err != nil {
			panic(fmt.Errorf("failed to create reapi fs: failed to calculate digest: %v", err))
		}
		directories[dg] = child
	}

	return &fs{
		c:           c,
		root:        tree.Root,
		directories: directories,
		workingDir:  workingDir,
	}
}

// Open opens the file with the given name
func (fs *fs) Open(name string) (iofs.File, error) {
	return fs.open(".", filepath.Join(fs.workingDir, name), fs.root)
}

func (fs *fs) open(path, name string, wd *pb.Directory) (iofs.File, error) {
	name, rest, hasToBeDir := strings.Cut(name, string(filepath.Separator))
	// Must be a dodgy symlink that goes past our tree.
	if name == ".." || name == "." {
		return nil, os.ErrNotExist
	}

	for _, d := range wd.Directories {
		if d.Name == name {
			dirPb := fs.directories[digest.NewFromProtoUnvalidated(d.Digest)]
			if rest == "" {
				i := &info{
					name:  name,
					isDir: true,
				}
				return &dir{
					info:     i.withProperties(dirPb.NodeProperties),
					pb:       dirPb,
					children: fs.directories,
				}, nil
			}
			return fs.open(filepath.Join(path, name), rest, dirPb)
		}
	}

	// If the path contains a /, we only resolve against dirs.
	if hasToBeDir {
		return nil, iofs.ErrNotExist
	}

	for _, f := range wd.Files {
		if f.Name == name {
			bs, _, err := fs.c.ReadBlob(context.Background(), digest.NewFromProtoUnvalidated(f.Digest))
			if err != nil {
				return nil, err
			}
			i := &info{
				isDir: false,
				size:  int64(len(bs)),
				name:  f.Name,
			}

			return &file{
				ReadSeeker: bytes.NewReader(bs),
				info:       i.withProperties(f.NodeProperties),
			}, nil
		}
	}

	for _, l := range wd.Symlinks {
		if l.Name == name {
			if filepath.IsAbs(l.Target) {
				// Some REAPI implementations support this, but this is considered invalid where Please is concerned.
				path = filepath.Join(path, name)
				return nil, fmt.Errorf("symlink %v is has abs target %v. This is not supported", path, l.Target)
			}
			ret, err := fs.Open(filepath.Join(path, l.Target))
			if err != nil {
				return nil, fmt.Errorf("failed to resolve symlink %v: %w", filepath.Join(path, l.Target), err)
			}
			return ret, nil
		}
	}
	return nil, iofs.ErrNotExist
}

type file struct {
	io.ReadSeeker
	*info
}

func (b *file) Stat() (iofs.FileInfo, error) {
	return b, nil
}
func (b *file) Close() error {
	return nil
}

type dir struct {
	pb       *pb.Directory
	children map[digest.Digest]*pb.Directory
	*info
}

// ReadDir is a slightly incorrect implementation of ReadDir. It deviates slightly as it will report all files have 0
// size. This seems to work for our limited purposes though.
func (p *dir) ReadDir(n int) ([]iofs.DirEntry, error) {
	dirSize := n
	if n <= 0 {
		dirSize = len(p.pb.Files) + len(p.pb.Symlinks) + len(p.pb.Files)
	}
	ret := make([]iofs.DirEntry, 0, dirSize)
	for _, dirNode := range p.pb.Directories {
		if n > 0 && len(ret) == n {
			return ret, nil
		}
		dir := p.children[digest.NewFromProtoUnvalidated(dirNode.Digest)]

		i := &info{
			name:     dirNode.Name,
			isDir:    true,
			typeMode: os.ModeDir,
		}

		i.withProperties(dir.NodeProperties)

		ret = append(ret, i)
	}
	for _, file := range p.pb.Files {
		if n > 0 && len(ret) == n {
			return ret, nil
		}
		i := &info{
			name: file.Name,
			// TODO(jpoole): technically we could calculate this on demand by allowing info.Size() to download the file
			// 	from the CAS... we don't need to for now though.
			size: 0,
		}
		i.withProperties(file.NodeProperties)
		ret = append(ret, i)
	}
	for _, link := range p.pb.Symlinks {
		if n > 0 && len(ret) == n {
			return ret, nil
		}
		i := &info{
			name:     link.Name,
			typeMode: os.ModeSymlink,
		}
		i.withProperties(link.NodeProperties)
		ret = append(ret, i)
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
