package plzinit

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/thought-machine/please/src/core"
)

func InitPleasings(location string, printOnly bool, revision string) error {
	if revision == "" {
		revision = "master"
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

	// TODO(jpoole): We could probably parse the file to update/append the rule
	f, err := os.OpenFile(location, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, pleasingsSubrepoTemplate, revision)
	return err
}
