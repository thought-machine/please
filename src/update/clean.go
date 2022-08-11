package update

import (
	"fmt"
	"os"
	"path"
	"sort"

	"github.com/coreos/go-semver/semver"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
)

// clean checks for any stale versions in the download directory and wipes them out if OK.
func clean(config *core.Configuration, manualUpdate bool) {
	dir, _ := os.ReadDir(config.Please.Location)
	versions := make(semver.Versions, 0, len(dir))
	// Convert these to semver
	for _, entry := range dir {
		if v, err := semver.NewVersion(entry.Name()); err == nil && !config.Please.Version.Equal(*v) {
			versions = append(versions, v)
		}
	}
	numToClean := len(versions) - config.Please.NumOldVersions
	if numToClean <= 0 {
		return
	} else if config.Please.Autoclean {
		log.Notice("Auto-cleaning old versions...")
	} else if cli.StdErrIsATerminal && manualUpdate { // Only prompt on `plz update`, otherwise it is annoying
		if !cli.PromptYN(fmt.Sprintf("Found %d old versions, will delete %d of them. OK?", len(versions), numToClean), true) {
			return
		}
	} else {
		// Not autoclean and no tty to prompt on.
		log.Warning("Found %d old versions, not cleaning due to autoclean = false", len(versions))
		return
	}
	// If we get here then we are a go for cleaning.
	sort.Sort(versions)
	for _, version := range versions[:numToClean] {
		log.Notice("Cleaning old version %s...", version)
		if err := os.RemoveAll(path.Join(config.Please.Location, version.String())); err != nil {
			log.Error("Couldn't remove %s: %s", version, err)
		}
	}
}
