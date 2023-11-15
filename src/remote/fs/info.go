package fs

import (
	iofs "io/fs"
	"os"
	"time"

	pb "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
)

// info represents information about a file/directory
type info struct {
	name    string
	size    int64
	modTime time.Time
	mode    os.FileMode
}

func newFileInfo(f *pb.FileNode) *info {
	i := info{
		size: f.Digest.SizeBytes,
		name: f.Name,
	}
	return i.withProperties(f.NodeProperties)
}
func newDirInfo(name string, dir *pb.Directory) *info {
	i := &info{
		name: name,
		mode: os.ModeDir,
	}
	return i.withProperties(dir.NodeProperties)
}

func newSymlinkInfo(node *pb.SymlinkNode) *info {
	i := &info{
		name: node.Name,
		mode: os.ModeSymlink,
	}
	return i.withProperties(node.NodeProperties)
}

// withProperties safely sets the node info if it's available.
func (i *info) withProperties(nodeProperties *pb.NodeProperties) *info {
	if nodeProperties == nil {
		return i
	}

	if nodeProperties.UnixMode != nil {
		// This should in theory have the type mode set already but we bitwise or here to to make sure this is preserved
		// from the constructors above in case the remote doesn't set this.
		i.mode |= os.FileMode(nodeProperties.UnixMode.Value)
	}

	if nodeProperties.Mtime != nil {
		i.modTime = nodeProperties.Mtime.AsTime()
	}
	return i
}

func (i *info) Type() iofs.FileMode {
	return i.mode.Type()
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
	return i.modTime
}

func (i *info) IsDir() bool {
	return i.mode.IsDir()
}

func (i *info) Sys() any {
	return nil
}
