// +build !windows

package fs

import "os"

// IsSameFile returns true if two filenames describe the same underlying file (i.e. inode)
func IsSameFile(a, b string) bool {
	i1, err1 := getInode(a)
	i2, err2 := getInode(b)
	return err1 == nil && err2 == nil && i1 == i2
}

// getInode returns the inode of a file.
func getInode(filename string) (uint64, error) {
	fi, err := os.Stat(filename)
	if err != nil {
		return 0, err
	}
	s, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("Not a syscall.Stat_t")
	}
	return s.Ino, nil
}