// +build windows

package fs

// IsSameFile returns true if two filenames describe the same underlying file (i.e. inode)
func IsSameFile(a, b string) bool {
	// TODO(jpoole): compare the equivalent of inodes on NTFS
	return a == b
}