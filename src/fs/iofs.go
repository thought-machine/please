package fs

import (
	iofs "io/fs"
	"os"
)

type osFS struct{}

func (osFS) ReadDir(name string) ([]iofs.DirEntry, error) {
	return os.ReadDir(name)
}

func (osFS) Open(name string) (iofs.File, error) {
	return os.Open(name)
}

// HostFS returns an io/fs.FS that behaves the same as the host OS i.e. the same way os.Open works.
var HostFS = osFS{}
