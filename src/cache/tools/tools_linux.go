package tools

import (
	"os"
	"syscall"
)

// AccessTime returns the last access time of a file.
func AccessTime(info os.FileInfo) int64 {
	return info.Sys().(*syscall.Stat_t).Atim.Sec
}
