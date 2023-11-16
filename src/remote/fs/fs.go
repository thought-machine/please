// Package fs provides an io/fs.FS implementation over the remote execution API content addressable store (CAS)
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

	"github.com/thought-machine/please/src/cli/logging"
)

var log = logging.Log

// Client is an interface to the REAPI CAS
type Client interface {
	ReadBlob(ctx context.Context, d digest.Digest) ([]byte, *client.MovedBytesMetadata, error)
}

func FindNode(fs iofs.FS, path string) (*pb.FileNode, *pb.DirectoryNode, *pb.SymlinkNode, error) {
	casFS, ok := fs.(*CASFileSystem)
	if !ok {
		return nil, nil, nil, fmt.Errorf("not supported")
	}
	return casFS.FindNode(path)
}

// CASFileSystem is an fs.FS implemented on top of a Tree proto. This will download files as they are needed from the
// CAS when they are opened.
type CASFileSystem struct {
	c           Client
	root        *pb.Directory
	directories map[digest.Digest]*pb.Directory
	workingDir  string
}

// Stat implements StatFS so that iofs.Stat doesn't download the file to determine this info.
func (fs *CASFileSystem) Stat(name string) (iofs.FileInfo, error) {
	file, dir, link, err := fs.FindNode(name)
	if err != nil {
		return nil, err
	}
	if file != nil {
		return newFileInfo(file), nil
	}
	if dir != nil {
		dirPB := fs.directories[digest.NewFromProtoUnvalidated(dir.Digest)]
		return newDirInfo(dir.Name, dirPB), nil
	}
	if link != nil {
		return newSymlinkInfo(link), nil
	}
	return nil, os.ErrNotExist
}

// New creates a new filesystem on top of the given proto, using client to download files from the CAS on demand.
func New(c Client, tree *pb.Tree, workingDir string) *CASFileSystem {
	directories := make(map[digest.Digest]*pb.Directory, len(tree.Children))
	for _, child := range append(tree.Children, tree.Root) {
		dg, err := digest.NewFromMessage(child)
		if err != nil {
			log.Fatalf("Failed to create CASFileSystem: failed to calculate digest: %v", err)
		}
		directories[dg] = child
	}

	return &CASFileSystem{
		c:           c,
		root:        tree.Root,
		directories: directories,
		workingDir:  filepath.Clean(workingDir),
	}
}

// Open opens the file with the given name
func (fs *CASFileSystem) Open(name string) (iofs.File, error) {
	return fs.open(filepath.Join(fs.workingDir, name))
}

// FindNode returns the node proto for the given name. Either FileNode, DirectoryNode or SymlinkNode will be set, or an
// error will be returned. The error will be os.ErrNotExist if the path doesn't exist.
func (fs *CASFileSystem) FindNode(name string) (*pb.FileNode, *pb.DirectoryNode, *pb.SymlinkNode, error) {
	return fs.findNode(fs.root, filepath.Join(fs.workingDir, name))
}

func (fs *CASFileSystem) open(name string) (iofs.File, error) {
	fileNode, dirNode, linkNode, err := fs.findNode(fs.root, name)
	if err != nil {
		return nil, err
	}

	if linkNode != nil {
		if filepath.IsAbs(linkNode.Target) {
			return nil, fmt.Errorf("%v: symlink target was absolute which is invalid", name)
		}
		return fs.open(filepath.Join(filepath.Dir(name), linkNode.Target))
	}

	if fileNode != nil {
		return fs.openFile(fileNode)
	}
	if dirNode != nil {
		return fs.openDir(dirNode)
	}
	return nil, os.ErrNotExist
}

// openFile downloads a file from the CAS and returns it as an iofs.File
func (fs *CASFileSystem) openFile(f *pb.FileNode) (*file, error) {
	bs, _, err := fs.c.ReadBlob(context.Background(), digest.NewFromProtoUnvalidated(f.Digest))
	if err != nil {
		return nil, err
	}

	return &file{
		ReadSeeker: bytes.NewReader(bs),
		info:       newFileInfo(f),
	}, nil
}

func (fs *CASFileSystem) openDir(d *pb.DirectoryNode) (iofs.File, error) {
	dirPb := fs.directories[digest.NewFromProtoUnvalidated(d.Digest)]

	return &dir{
		info:     newDirInfo(d.Name, dirPb),
		pb:       dirPb,
		children: fs.directories,
	}, nil
}

func (fs *CASFileSystem) findNode(wd *pb.Directory, name string) (*pb.FileNode, *pb.DirectoryNode, *pb.SymlinkNode, error) {
	// When the path contains a /, we only want to match name as a directory. This is because if we have foo/bar, and we
	// matched foo as a file, we still need to descend further, which we can't do if it's a file or symlink.
	name, rest, hasToBeDir := strings.Cut(name, string(filepath.Separator))

	if name == "." {
		if rest != "" {
			return fs.findNode(wd, rest)
		}
		dg, err := digest.NewFromMessage(wd)
		if err != nil {
			return nil, nil, nil, err
		}
		node := &pb.DirectoryNode{Name: ".", Digest: dg.ToProto()}
		return nil, node, nil, nil
	}

	// Must be a dodgy symlink that goes past our tree.
	if name == ".." {
		return nil, nil, nil, os.ErrNotExist
	}

	for _, d := range wd.Directories {
		if d.Name == name {
			dirPb := fs.directories[digest.NewFromProtoUnvalidated(d.Digest)]
			if rest == "" {
				return nil, d, nil, nil
			}
			return fs.findNode(dirPb, rest)
		}
	}

	if hasToBeDir {
		return nil, nil, nil, os.ErrNotExist
	}

	for _, f := range wd.Files {
		if f.Name == name {
			return f, nil, nil, nil
		}
	}

	for _, l := range wd.Symlinks {
		if l.Name == name {
			return nil, nil, l, nil
		}
	}
	return nil, nil, nil, os.ErrNotExist
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

// ReadDir implements listing the contents of a directory stored in the CAS. This is entirely based off the original
// data from the Tree proto so doesn't do any additional fetching.
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
		ret = append(ret, newDirInfo(dirNode.Name, dir))
	}
	for _, file := range p.pb.Files {
		if n > 0 && len(ret) == n {
			return ret, nil
		}

		ret = append(ret, newFileInfo(file))
	}
	for _, link := range p.pb.Symlinks {
		if n > 0 && len(ret) == n {
			return ret, nil
		}
		ret = append(ret, newSymlinkInfo(link))
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
