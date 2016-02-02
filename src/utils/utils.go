// Package utils contains various utility functions and whatnot.
package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/Songmu/prompter"
	"github.com/op/go-logging"

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
}
