package plzinit

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

func InitPleasings(location string, printOnly bool, revision string) error {
	if !printOnly && fs.FileExists(location) {
		if !cli.PromptYN(fmt.Sprintf("It looks like a build file already exists at %v. You may use --print to print the rule and add it manually instead. Override BUILD", location), false) {
			return nil
		}
	}

	if printOnly {
		fmt.Printf(pleasingsSubrepoTemplate, revision)
		return nil
	}

	dir := filepath.Dir(location)
	if dir != "." {
		if err := os.MkdirAll(dir, core.DirPermissions); err != nil {
			log.Fatalf("failed to create pleasings directory: %v", err)
		}
	}

	// TODO(jpoole): We could probably parse the file, update/append the rule, and re-serialise that rather than nuking it
	if err := os.RemoveAll(location); err != nil {
		return err
	}
	return ioutil.WriteFile(location, []byte(fmt.Sprintf(pleasingsSubrepoTemplate, revision)), 0644)
}
