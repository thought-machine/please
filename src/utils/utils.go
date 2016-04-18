// Package utils contains various utility functions and whatnot.
package utils

import (
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
const wrapperScriptName = "pleasew"

// InitConfig initialises a .plzconfig template in the given directory.
func InitConfig(dir string) {
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
	if core.FileExists(config) && !prompter.YN(fmt.Sprintf("Would create %s but it already exists. This will wipe out any previous config in the file - continue?", config), false) {
		os.Exit(1)
	}
	contents := fmt.Sprintf(configTemplate, core.PleaseVersion)
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
			} else if name == "plz-out" || (info.IsDir() && strings.HasPrefix(info.Name(), ".") && name != ".") {
				return filepath.SkipDir // Don't walk output or hidden directories
			} else if info.IsDir() && !strings.HasPrefix(name, prefix) && !strings.HasPrefix(prefix, name) {
				return filepath.SkipDir // Skip any directory without the prefix we're after (but not any directory beneath that)
			} else if isABuildFile(info.Name(), config) && !info.IsDir() {
				dir, _ := path.Split(name)
				ch <- strings.TrimRight(dir, "/")
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

// isABuildFile returns true if given filename is a build file name.
func isABuildFile(name string, config *core.Configuration) bool {
	for _, buildFileName := range config.Please.BuildFileName {
		if name == buildFileName {
			return true
		}
	}
	return false
}
