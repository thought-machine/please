// +build !bootstrap

package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Songmu/prompter"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

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
		if core.FindRepoRoot() {
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

// InitConfigFile sets a bunch of values in a config file.
func InitConfigFile(filename string, options map[string]string) {
	b := readConfig(filename)
	for k, v := range options {
		parts := strings.Split(k, ".")
		if len(parts) != 2 {
			log.Fatalf("unknown key format: %s", k)
		}
		b = append(b, []byte(fmt.Sprintf("[%s]\n%s = %s\n", parts[0], parts[1], v))...)
	}
	if err := fs.EnsureDir(filename); err != nil {
		log.Fatalf("Cannot create directory for new file: %s", err)
	} else if err := ioutil.WriteFile(filename, b, 0644); err != nil {
		log.Fatalf("Failed to write updated config file: %s", err)
	}
}

func readConfig(filename string) []byte {
	if !fs.PathExists(filename) {
		return nil
	}
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("Failed to read config file: %s", err)
	}
	return b
}
