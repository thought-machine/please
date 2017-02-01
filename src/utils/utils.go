// Package utils contains various utility functions and whatnot.
package utils

import (
	"bufio"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/op/go-logging.v1"

	"core"
)

var log = logging.MustGetLogger("utils")

// Finds all packages under a particular path.
// Used to implement rules with ... where we need to know all possible packages
// under that location.
func FindAllSubpackages(config *core.Configuration, rootPath string, prefix string) <-chan string {
	ch := make(chan string)
	go func() {
		if rootPath == "" {
			rootPath = "."
		}
		if err := filepath.Walk(rootPath, func(name string, info os.FileInfo, err error) error {
			if err != nil {
				return err // stop on any error
			} else if name == core.OutDir || (info.IsDir() && strings.HasPrefix(info.Name(), ".") && name != ".") {
				return filepath.SkipDir // Don't walk output or hidden directories
			} else if info.IsDir() && !strings.HasPrefix(name, prefix) && !strings.HasPrefix(prefix, name) {
				return filepath.SkipDir // Skip any directory without the prefix we're after (but not any directory beneath that)
			} else if isABuildFile(info.Name(), config) && !info.IsDir() {
				dir, _ := path.Split(name)
				ch <- strings.TrimRight(dir, "/")
			} else if name == config.Please.ExperimentalDir {
				return filepath.SkipDir // Skip the experimental directory if it's set
			}
			// Check against blacklist
			for _, dir := range config.Please.BlacklistDirs {
				if dir == info.Name() {
					return filepath.SkipDir
				}
			}
			return nil
		}); err != nil {
			log.Fatalf("Failed to walk tree under %s; %s\n", rootPath, err)
		}
		close(ch)
	}()
	return ch
}

var seenStdin = false // Used to track that we don't try to read stdin twice

// isABuildFile returns true if given filename is a build file name.
func isABuildFile(name string, config *core.Configuration) bool {
	for _, buildFileName := range config.Please.BuildFileName {
		if name == buildFileName {
			return true
		}
	}
	return false
}

// ReadStdin reads a sequence of space-delimited words from standard input.
// Words are pushed onto the returned channel asynchronously.
func ReadStdin() <-chan string {
	c := make(chan string)
	if seenStdin {
		log.Fatalf("Repeated - on command line; can't reread stdin.")
	}
	seenStdin = true
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Split(bufio.ScanWords)
		for scanner.Scan() {
			s := strings.TrimSpace(scanner.Text())
			if s != "" {
				c <- s
			}
		}
		if err := scanner.Err(); err != nil {
			log.Fatalf("Error reading stdin: %s", err)
		}
		close(c)
	}()
	return c
}

// ReadAllStdin reads standard input in its entirety to a slice.
// Since this reads it completely before returning it won't handle a slow input
// very nicely. ReadStdin is therefore preferable when possible.
func ReadAllStdin() []string {
	var ret []string
	for s := range ReadStdin() {
		ret = append(ret, s)
	}
	return ret
}
