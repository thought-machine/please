package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/update"
)

func main() {
	cli.InitLogging(cli.Verbosity(logging.WARNING))

	core.MustFindRepoRoot()
	if err := os.Chdir(core.RepoRoot); err != nil {
		log.Fatalf("Error moving to '%s' repo root: %s", core.RepoRoot, err)
	}

	config, err := core.ReadDefaultConfigFiles(nil)
	if err != nil {
		log.Fatalf("Error reading config file: %s", err)
	}

	plzPath := filepath.Join(config.Please.Location, "please")

	if fs.FileExists(plzPath) {
		out, err := exec.Command(plzPath, "--version").Output()
		if err != nil {
			log.Fatalf("Error getting Please version: %s", err)
		}

		core.PleaseVersion = strings.TrimSpace(strings.TrimPrefix(string(out), "Please version "))
	}

	update.CheckAndUpdate(config, true, false, false, true, true, false)

	if err := syscall.Exec(plzPath, append(os.Args, "--noupdate"), os.Environ()); err != nil {
		log.Fatalf("Failed to execute Please: %s", err)
	}
}
