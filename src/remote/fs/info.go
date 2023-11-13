package fs

import (
	iofs "io/fs"
	"os"
	"time"
)

// info represents information about a file/directory
type info struct {
	name     string
	isDir    bool
	size     int64
	modTime  time.Time
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
	return i.modTime
}

func (i *info) IsDir() bool {
	return i.isDir
}

func (i *info) Sys() any {
	return nil
}
