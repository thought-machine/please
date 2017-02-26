// Small program to clean directory cache.
// This is actually fairly tricky since Please doesn't have a daemon managing it;
// we don't particularly want that so instead we fire off one of these processes
// at each invocation of 'build' or 'test' to check the cache size and clean out stuff
// if it looks too big.

package main

import (
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/djherbis/atime"
	"github.com/dustin/go-humanize"
	"gopkg.in/op/go-logging.v1"

	"cli"
)

var log = logging.MustGetLogger("cache_cleaner")

// Period of time in seconds between which two artifacts are considered to have the same atime.
const accessTimeGracePeriod = 600 // Ten minutes

type CacheEntry struct {
	Path  string
	Size  int64
	Atime int64
}
type CacheEntries []CacheEntry

func (entries CacheEntries) Len() int      { return len(entries) }
func (entries CacheEntries) Swap(i, j int) { entries[i], entries[j] = entries[j], entries[i] }
func (entries CacheEntries) Less(i, j int) bool {
	diff := entries[i].Atime - entries[j].Atime
	if diff > -accessTimeGracePeriod && diff < accessTimeGracePeriod {
		return entries[i].Size > entries[j].Size
	}
	return entries[i].Atime < entries[j].Atime
}

func findSize(path string) (int64, error) {
	var totalSize int64 = 0
	if err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		} else {
			totalSize += info.Size()
			return nil
		}
	}); err != nil {
		return 0, err
	} else {
		return totalSize, nil
	}
}

func start(directory string, highWaterMark, lowWaterMark int64) {
	entries := CacheEntries{}
	var totalSize int64 = 0
	if err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		} else if (len(info.Name()) == 28 || len(info.Name()) == 29) && info.Name()[27] == '=' {
			// Directory has the right length. We do this in an attempt to clean only entire
			// entries in the cache, not just individual files from them.
			// 28 == length of 20-byte sha1 hash, encoded to base64, which always gets a trailing =
			// as padding so we can check that to be "sure".
			// Also 29 in case we appended an extra = (see below)
			if size, err := findSize(path); err != nil {
				return err
			} else {
				entries = append(entries, CacheEntry{path, size, atime.Get(info).Unix()})
				totalSize += size
				return filepath.SkipDir
			}
		} else {
			return nil // nothing particularly to do for other entries
		}
	}); err != nil {
		log.Fatalf("error walking cache directory: %s\n", err)
	}
	log.Notice("Total cache size: %s", humanize.Bytes(uint64(totalSize)))
	if totalSize < highWaterMark {
		return // Nothing to do, cache is small enough.
	}
	// OK, we need to slim it down a bit. We implement a simple LRU algorithm.
	sort.Sort(entries)
	for _, entry := range entries {
		log.Notice("Cleaning %s, accessed %s, saves %s", entry.Path, humanize.Time(time.Unix(entry.Atime, 0)), humanize.Bytes(uint64(entry.Size)))
		// Try to rename the directory first so we don't delete bits while someone might access them.
		newPath := entry.Path + "="
		if err := os.Rename(entry.Path, newPath); err != nil {
			log.Errorf("Couldn't rename %s: %s", entry.Path, err)
			continue
		}
		if err := os.RemoveAll(newPath); err != nil {
			log.Errorf("Couldn't remove %s: %s", newPath, err)
			continue
		}
		totalSize -= entry.Size
		if totalSize < lowWaterMark {
			break
		}
	}
}

var opts struct {
	Verbosity     int          `short:"v" long:"verbosity" description:"Verbosity of output (higher number = more output, default 2 -> notice, warnings and errors only)" default:"2"`
	LowWaterMark  cli.ByteSize `short:"l" long:"low_water_mark" description:"Size of cache to clean down to" default:"8G"`
	HighWaterMark cli.ByteSize `short:"i" long:"high_water_mark" description:"Max size of cache to clean at" default:"10G"`
	Directory     string       `short:"d" long:"dir" required:"true" description:"Location of cache directory"`
}

func main() {
	cli.ParseFlagsOrDie("Please directory cache cleaner", "5.5.0", &opts)
	cli.InitLogging(opts.Verbosity)
	start(opts.Directory, int64(opts.HighWaterMark), int64(opts.LowWaterMark))
	os.Exit(0)
}
