// Code for Please auto-updating itself.
// At startup, Please can check a version set in the config file. If that doesn't
// match the version of the current binary, it will download the appropriate
// version from the website and swap to using that instead.
//
// This feature is fairly directly cribbed from Buck since we found it very useful,
// albeit implemented differently so it plays nicer with multiple simultaneous
// builds on the same machine.

package update

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"path"
	"runtime"
	"strings"
	"syscall"

	"gopkg.in/op/go-logging.v1"

	"core"
)

var log = logging.MustGetLogger("update")

func CheckAndUpdate(config *core.Configuration, shouldUpdate, forceUpdate bool) {
	if config.Please.Version == core.PleaseVersion {
		return // Version matches, nothing to do here.
	} else if (!shouldUpdate || !config.Please.SelfUpdate) && !forceUpdate {
		log.Warning("Update to Please version %s skipped (current version: %s)", config.Please.Version, core.PleaseVersion)
		return
	} else if config.Please.Location == "" {
		log.Warning("Please location not set in config, cannot auto-update.")
		return
	} else if config.Please.DownloadLocation == "" {
		log.Warning("Please download location not set in config, cannot auto-update.")
		return
	}
	if config.Please.Version == "" {
		if !forceUpdate {
			config.Please.Version = core.PleaseVersion
			return
		}
		config.Please.Version = findLatestVersion(config)
		CheckAndUpdate(config, shouldUpdate, forceUpdate)
		return
	}

	// Okay, now we're past all that...
	log.Warning("Updating to Please version %s (currently %s)", config.Please.Version, core.PleaseVersion)

	// Must lock here so that the update process doesn't race when running two instances
	// simultaneously.
	core.AcquireRepoLock()
	defer core.ReleaseRepoLock()

	newPlease := path.Join(config.Please.Location, config.Please.Version, "please")
	if !core.PathExists(newPlease) {
		downloadPlease(config)
	}
	linkNewPlease(config)
	args := append([]string{newPlease}, "--assert_version", config.Please.Version)
	args = append(args, os.Args[1:]...)
	log.Info("Executing %s", strings.Join(args, " "))
	if err := syscall.Exec(newPlease, args, os.Environ()); err != nil {
		log.Fatalf("Failed to exec new Please version %s: %s", newPlease, err)
	}
	// Shouldn't ever get here. We should have either exec'd or died above.
	panic("please update failed in an an unexpected and exciting way")
}

func downloadPlease(config *core.Configuration) {
	newDir := path.Join(config.Please.Location, config.Please.Version)
	if err := os.MkdirAll(newDir, core.DirPermissions); err != nil {
		log.Fatalf("Failed to create directory %s: %s", newDir, err)
	}

	// Make sure from here on that we don't leave partial directories hanging about.
	// If someone ctrl+C's during this download then on re-running we might
	// have partial files written there that don't really work.
	defer func() {
		if r := recover(); r != nil {
			cleanDir(newDir)
			log.Fatalf("Failed to download Please: %s", r)
		}
	}()
	go handleSignals(newDir)
	mustClose := func(closer io.Closer) {
		if err := closer.Close(); err != nil {
			panic(err)
		}
	}

	url := fmt.Sprintf("%s/%s_%s/%s/please_%s.tar.gz", config.Please.DownloadLocation, runtime.GOOS, runtime.GOARCH, config.Please.Version, config.Please.Version)
	log.Info("Downloading %s", url)
	response, err := http.Get(url)
	if err != nil {
		panic(fmt.Sprintf("Failed to download %s: %s", url, err))
	} else if response.StatusCode < 200 || response.StatusCode > 299 {
		panic(fmt.Sprintf("Failed to download %s: got response %s", url, response.Status))
	}
	defer mustClose(response.Body)

	gzreader, err := gzip.NewReader(response.Body)
	if err != nil {
		panic(fmt.Sprintf("%s isn't a valid gzip file: %s", url, err))
	}
	defer mustClose(gzreader)

	tarball := tar.NewReader(gzreader)
	for {
		hdr, err := tarball.Next()
		if err == io.EOF {
			break // End of archive
		} else if err != nil {
			panic(fmt.Sprintf("Error un-tarring %s: %s", url, err))
		}
		filename := path.Base(hdr.Name)
		destination := path.Join(newDir, filename)
		log.Info("Extracting %s to %s", filename, destination)
		if contents, err := ioutil.ReadAll(tarball); err != nil {
			panic(fmt.Sprintf("Error extracting %s from tarball: %s", filename, err))
		} else if err := ioutil.WriteFile(destination, contents, fileMode(filename)); err != nil {
			panic(fmt.Sprintf("Failed to write to %s: %s", destination, err))
		}
	}
}

func linkNewPlease(config *core.Configuration) {
	if files, err := ioutil.ReadDir(path.Join(config.Please.Location, config.Please.Version)); err != nil {
		log.Fatalf("Failed to read directory: %s", err)
	} else {
		for _, file := range files {
			linkNewFile(config, file.Name())
		}
	}
}

func linkNewFile(config *core.Configuration, file string) {
	newDir := path.Join(config.Please.Location, config.Please.Version)
	globalFile := path.Join(config.Please.Location, file)
	downloadedFile := path.Join(newDir, file)
	if err := os.RemoveAll(globalFile); err != nil {
		log.Fatalf("Failed to remove existing file %s: %s", globalFile, err)
	}
	if err := os.Symlink(downloadedFile, globalFile); err != nil {
		log.Fatalf("Error linking %s -> %s: %s", downloadedFile, globalFile, err)
	}
	log.Info("Linked %s -> %s", globalFile, downloadedFile)
}

func fileMode(filename string) os.FileMode {
	if strings.HasSuffix(filename, ".jar") {
		return 0664 // The .jar files obviously aren't executable
	} else {
		return 0775 // Everything else we download is.
	}
}

func cleanDir(newDir string) {
	log.Notice("Attempting to clean directory %s", newDir)
	if err := os.RemoveAll(newDir); err != nil {
		log.Errorf("Failed to clean %s: %s", newDir, err)
	}
}

// handleSignals traps SIGINT and SIGKILL (if possible) and on receiving one cleans the given directory.
func handleSignals(newDir string) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	s := <-c
	log.Notice("Got signal %s", s)
	cleanDir(newDir)
	log.Fatalf("Got signal %s", s)
}

// findLatestVersion attempts to find the latest available version of plz.
func findLatestVersion(config *core.Configuration) string {
	url := config.Please.DownloadLocation + "/latest_version"
	log.Info("Downloading %s", url)
	response, err := http.Get(url)
	if err != nil {
		log.Fatalf("Failed to find latest plz version: %s", err)
	}
	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("Failed to find latest plz version: %s", err)
	}
	return strings.TrimSpace(string(data))
}
