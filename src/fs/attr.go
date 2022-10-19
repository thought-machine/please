package fs

import (
	"os"
	"path/filepath"

	"github.com/pkg/xattr"
)

// RecordAttr records an attribute on the given file, using xattrs if available, otherwise
// falling back to writing a separate file.
func RecordAttr(filename string, hash []byte, xattrName string, xattrsEnabled bool) error {
	if !xattrsEnabled {
		return RecordAttrFile(filename, hash)
	}
	if err := xattr.LSet(filename, xattrName, hash); err != nil {
		if IsSymlink(filename) {
			// On Linux at least, symlinks don't accept hashes.
			return RecordAttrFile(filename, hash)
		} else if os.IsPermission(err.(*xattr.Error).Err) {
			// Can't set xattrs without write permission... attempt to cheekily chmod it first.
			if info, err := os.Lstat(filename); err == nil {
				if err := os.Chmod(filename, info.Mode()|0200); err == nil {
					defer os.Chmod(filename, info.Mode())
					return xattr.LSet(filename, xattrName, hash)
				}
			}
		}
		return err
	}
	return nil
}

// RecordAttrFile records a hash for the given file. It's the fallback for RecordAttr when
// xattrs aren't available.
// The actual filename written will differ from the original (since obviously we cannot overwrite it).
func RecordAttrFile(filename string, hash []byte) error {
	return os.WriteFile(fallbackFileName(filename), hash, 0644)
}

func fallbackFileName(filename string) string {
	dir, file := filepath.Split(filename)
	return dir + ".rule_hash_" + file
}

// ReadAttr reads an attribute from the given file, using xattrs if available, otherwise
// falling back to reading a separate file.
// It returns an empty slice if it can't be read.
func ReadAttr(filename, xattrName string, xattrsEnabled bool) []byte {
	if !xattrsEnabled {
		return ReadAttrFile(filename)
	}
	b, err := xattr.LGet(filename, xattrName)
	if err != nil {
		if IsSymlink(filename) {
			// Symlinks can't take xattrs on Linux. We stash it on the fallback hash file instead.
			return ReadAttrFile(filename)
		} else if e2 := err.(*xattr.Error).Err; !os.IsNotExist(e2) && e2 != xattr.ENOATTR {
			log.Warning("Failed to read hash for %s: %s", filename, err)
		}
		return nil
	}
	return b
}

// ReadAttrFile reads a hash for the given file. It's the fallback for ReadAttr and pairs with
// RecordAttrFile to read the same files it would write.
func ReadAttrFile(filename string) []byte {
	b, _ := os.ReadFile(fallbackFileName(filename))
	return b
}
