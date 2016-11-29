// Package utils contains various utility functions and whatnot.
package utils

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Songmu/prompter"
	"gopkg.in/op/go-logging.v1"

	"core"
)

var log = logging.MustGetLogger("utils")

const configTemplate = `; Please config file
; Leaving this file as is is enough to use plz to build your project.
; Please will stay on whatever version you currently have until you run
; 'plz update', when it will download the latest available version.
;
; Or you can uncomment the following to pin everyone to a particular version;
; when you change it all users will automatically get updated.
; [please]
; version = %s
`
const bazelCompatibilityConfig = `
[bazel]
compatibility = true
`
const wrapperScriptName = "pleasew"

// InitConfig initialises a .plzconfig template in the given directory.
func InitConfig(dir string, bazelCompatibility bool) {
	if dir == "." {
		core.FindRepoRoot(false)
		if core.RepoRoot != "" {
			config := path.Join(core.RepoRoot, core.ConfigFileName)
			if !prompter.YN(fmt.Sprintf("You already seem to be in a plz repo (found %s). Continue?", config), false) {
				os.Exit(1)
			}
		}
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		log.Warning("Can't determine absolute directory: %s", err)
	}
	config := path.Join(dir, core.ConfigFileName)
	contents := fmt.Sprintf(configTemplate, core.PleaseVersion)
	if bazelCompatibility {
		contents += bazelCompatibilityConfig
	}
	if err := ioutil.WriteFile(config, []byte(contents), 0644); err != nil {
		log.Fatalf("Failed to write file: %s", err)
	}
	fmt.Printf("Wrote config template to %s, you're now ready to go!\n", config)
	// Now write the wrapper script
	data := MustAsset(wrapperScriptName)
	if err := ioutil.WriteFile(wrapperScriptName, data, 0755); err != nil {
		log.Fatalf("Failed to write file: %s", err)
	}
	fmt.Printf("\nAlso wrote wrapper script to %s; users can invoke that directly to run Please, even without it installed.\n", wrapperScriptName)
}

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
