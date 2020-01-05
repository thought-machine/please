// +build !bootstrap

// Package update contains code for Please auto-updating itself.
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
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/coreos/go-semver/semver"
	"github.com/ulikunitz/xz"
	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/process"
)

var log = logging.MustGetLogger("update")

// minSignedVersion is the earliest version of Please that has a signature.
var minSignedVersion = semver.Version{Major: 9, Minor: 2}

var httpClient http.Client

// CheckAndUpdate checks whether we should update Please and does so if needed.
// If it requires an update it will never return, it will either die on failure or on success will exec the new Please.
// Conversely, if an update isn't required it will return. It may adjust the version in the configuration.
// updatesEnabled indicates whether updates are enabled (i.e. not run with --noupdate)
// updateCommand indicates whether an update is specifically requested (due to e.g. `plz update`)
// forceUpdate indicates whether the user passed --force on the command line, in which case we
// will always update even if the version exists.
func CheckAndUpdate(config *core.Configuration, updatesEnabled, updateCommand, forceUpdate, verify bool) {
	if !shouldUpdate(config, updatesEnabled, updateCommand) && !forceUpdate {
		clean(config, updateCommand)
		return
	}
	word := describe(config.Please.Version.Semver(), core.PleaseVersion, true)
	if !updateCommand {
		log.Warning("%s to Please version %s (currently %s)", word, config.Please.Version.VersionString(), core.PleaseVersion)
	}

	// Must lock here so that the update process doesn't race when running two instances
	// simultaneously.
	core.AcquireRepoLock(nil)
	defer core.ReleaseRepoLock()

	// If the destination exists and the user passed --force, remove it to force a redownload.
	newDir := core.ExpandHomePath(path.Join(config.Please.Location, config.Please.Version.VersionString()))
	if forceUpdate && core.PathExists(newDir) {
		if err := os.RemoveAll(newDir); err != nil {
			log.Fatalf("Failed to remove existing directory: %s", err)
		}
	}

	// Honour the proxy setting if it's in the config.
	if config.Build.HTTPProxy != "" {
		httpClient.Transport = &http.Transport{
			Proxy: http.ProxyURL(config.Build.HTTPProxy.AsURL()),
		}
	}

	// Download it.
	newPlease := downloadAndLinkPlease(config, verify)

	// Clean out any old ones
	clean(config, updateCommand)

	// Now run the new one.
	core.ReturnToInitialWorkingDir()
	args := filterArgs(forceUpdate, append([]string{newPlease}, os.Args[1:]...))
	log.Info("Executing %s", strings.Join(args, " "))
	if err := syscall.Exec(newPlease, args, os.Environ()); err != nil {
		log.Fatalf("Failed to exec new Please version %s: %s", newPlease, err)
	}
	// Shouldn't ever get here. We should have either exec'd or died above.
	panic("please update failed in an an unexpected and exciting way")
}

// shouldUpdate determines whether we should run an update or not. It returns true iff one is required.
func shouldUpdate(config *core.Configuration, updatesEnabled, updateCommand bool) bool {
	if config.Please.Version.Semver() == core.PleaseVersion {
		return false // Version matches, nothing to do here.
	} else if config.Please.Version.IsGTE && config.Please.Version.LessThan(core.PleaseVersion) {
		if !updateCommand {
			return false // Version specified is >= and we are above it, nothing to do unless it's `plz update`
		}
		// Find the latest available version. Update if it's newer than the current one.
		config.Please.Version = *findLatestVersion(config.Please.DownloadLocation.String())
		return config.Please.Version.Semver() != core.PleaseVersion
	} else if (!updatesEnabled || !config.Please.SelfUpdate) && !updateCommand {
		// Update is required but has been skipped (--noupdate or whatever)
		if config.Please.Version.Major != 0 {
			word := describe(config.Please.Version.Semver(), core.PleaseVersion, true)
			log.Warning("%s to Please version %s skipped (current version: %s)", word, config.Please.Version, core.PleaseVersion)
		}
		return false
	} else if config.Please.Location == "" {
		log.Warning("Please location not set in config, cannot auto-update.")
		return false
	} else if config.Please.DownloadLocation == "" {
		log.Warning("Please download location not set in config, cannot auto-update.")
		return false
	}
	if config.Please.Version.Major == 0 {
		// Specific version isn't set, only update on `plz update`.
		if !updateCommand {
			config.Please.Version.Set(core.PleaseVersion.String())
			return false
		}
		config.Please.Version = *findLatestVersion(config.Please.DownloadLocation.String())
		return shouldUpdate(config, updatesEnabled, updateCommand)
	}
	return true
}

// downloadAndLinkPlease downloads a new Please version and links it into place, if needed.
// It returns the new location and dies on failure.
func downloadAndLinkPlease(config *core.Configuration, verify bool) string {
	config.Please.Location = core.ExpandHomePath(config.Please.Location)
	newPlease := path.Join(config.Please.Location, config.Please.Version.VersionString(), "please")

	if !core.PathExists(newPlease) {
		downloadPlease(config, verify)
	}
	if !verifyNewPlease(newPlease, config.Please.Version.VersionString()) {
		cleanDir(path.Join(config.Please.Location, config.Please.Version.VersionString()))
		log.Fatalf("Not continuing.")
	}
	linkNewPlease(config)
	return newPlease
}

func downloadPlease(config *core.Configuration, verify bool) {
	newDir := path.Join(config.Please.Location, config.Please.Version.VersionString())
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
	cli.AtExit(func() {
		cleanDir(newDir)
	})
	mustClose := func(closer io.Closer) {
		if err := closer.Close(); err != nil {
			panic(err)
		}
	}

	url := strings.TrimSuffix(config.Please.DownloadLocation.String(), "/")
	ext := "gz"
	if shouldUseXZ(config.Please.Version) {
		ext = "xz"
	}
	v := config.Please.Version.VersionString()
	url = fmt.Sprintf("%s/%s_%s/%s/please_%s.tar.%s", url, runtime.GOOS, runtime.GOARCH, v, v, ext)
	rc := mustDownload(url, true)
	defer mustClose(rc)
	var r io.Reader = bufio.NewReader(rc)

	if len(config.Please.VersionChecksum) > 0 {
		r = mustVerifyHash(r, config.Please.VersionChecksum)
	}

	if verify && config.Please.Version.LessThan(minSignedVersion) {
		log.Warning("Won't verify signature of download, version is too old to be signed.")
	} else if verify {
		r = verifyDownload(r, url)
	} else {
		log.Warning("Signature verification disabled for %s", url)
	}

	if shouldUseXZ(config.Please.Version) {
		xzr, err := xz.NewReader(r)
		if err != nil {
			panic(fmt.Sprintf("%s isn't a valid xzip file: %s", url, err))
		}
		copyTarFile(xzr, newDir, url)
	} else {
		gzreader, err := gzip.NewReader(r)
		if err != nil {
			panic(fmt.Sprintf("%s isn't a valid gzip file: %s", url, err))
		}
		defer mustClose(gzreader)
		copyTarFile(gzreader, newDir, url)
	}
}

func copyTarFile(zr io.Reader, newDir, url string) {
	tarball := tar.NewReader(zr)
	for {
		hdr, err := tarball.Next()
		if err == io.EOF {
			break // End of archive
		} else if err != nil {
			panic(fmt.Sprintf("Error un-tarring %s: %s", url, err))
		} else if err := writeTarFile(hdr, tarball, newDir); err != nil {
			panic(err)
		}
	}
}

// mustDownload downloads the contents of the given URL and returns its body
// The caller must close the reader when done.
// It panics if the download fails.
func mustDownload(url string, progress bool) io.ReadCloser {
	log.Info("Downloading %s", url)
	response, err := httpClient.Get(url)
	if err != nil {
		panic(fmt.Sprintf("Failed to download %s: %s", url, err))
	} else if response.StatusCode < 200 || response.StatusCode > 299 {
		panic(fmt.Sprintf("Failed to download %s: got response %s", url, response.Status))
	} else if progress {
		i, _ := strconv.Atoi(response.Header.Get("Content-Length"))
		return cli.NewProgressReader(response.Body, i, "Downloading")
	}
	return response.Body
}

func linkNewPlease(config *core.Configuration) {
	if files, err := ioutil.ReadDir(path.Join(config.Please.Location, config.Please.Version.VersionString())); err != nil {
		log.Fatalf("Failed to read directory: %s", err)
	} else {
		for _, file := range files {
			linkNewFile(config, file.Name())
		}
	}
}

func linkNewFile(config *core.Configuration, file string) {
	newDir := path.Join(config.Please.Location, config.Please.Version.VersionString())
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
	if strings.HasSuffix(filename, ".jar") || strings.HasSuffix(filename, ".so") {
		return 0664 // The .jar files obviously aren't executable
	}
	return 0775 // Everything else we download is.
}

func cleanDir(newDir string) {
	log.Notice("Attempting to clean directory %s", newDir)
	if err := os.RemoveAll(newDir); err != nil {
		log.Errorf("Failed to clean %s: %s", newDir, err)
	}
}

// findLatestVersion attempts to find the latest available version of plz.
func findLatestVersion(downloadLocation string) *cli.Version {
	url := strings.TrimRight(downloadLocation, "/") + "/latest_version"
	response := mustDownload(url, false)
	defer response.Close()
	data, err := ioutil.ReadAll(response)
	if err != nil {
		log.Fatalf("Failed to find latest plz version: %s", err)
	}
	return cli.MustNewVersion(strings.TrimSpace(string(data)))
}

// describe returns a word describing the process we're about to do ("update", "downgrading", etc)
func describe(a, b semver.Version, verb bool) string {
	if verb && a.LessThan(b) {
		return "Downgrading"
	} else if verb {
		return "Upgrading"
	} else if a.LessThan(b) {
		return "Downgrade"
	}
	return "Upgrade"
}

// verifyNewPlease calls a newly downloaded Please version to verify it's the expected version.
// It returns true iff the version is as expected.
func verifyNewPlease(newPlease, version string) bool {
	version = "Please version " + version // Output is prefixed with this.
	output, err := process.ExecCommand(newPlease, "--version")
	if err != nil {
		log.Errorf("Failed to run new Please: %s", err)
		return false
	}
	if strings.TrimSpace(string(output)) != version {
		log.Errorf("Bad version of Please downloaded: expected %s, but it's actually %s", version, string(output))
		return false
	}
	return true
}

// writeTarFile writes a file from a tarball to the filesystem in the corresponding location.
func writeTarFile(hdr *tar.Header, r io.Reader, destination string) error {
	// Strip the first directory component in the tarball

	stripped := hdr.Name[strings.IndexRune(hdr.Name, os.PathSeparator)+1:]
	dest := path.Join(destination, stripped)
	if err := os.MkdirAll(path.Dir(dest), core.DirPermissions); err != nil {
		return fmt.Errorf("Can't make destination directory: %s", err)
	}
	// Handle symlinks, but not other non-file things.
	if hdr.Typeflag == tar.TypeSymlink {
		return os.Symlink(hdr.Linkname, dest)
	} else if hdr.Typeflag != tar.TypeReg {
		return nil // Don't write directory entries, or rely on them being present.
	}
	log.Info("Extracting %s to %s", hdr.Name, dest)
	f, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE, os.FileMode(hdr.Mode))
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

// filterArgs filters out the --force update if forced updates were specified.
// This is important so that we don't end up in a loop of repeatedly forcing re-downloads.
func filterArgs(forceUpdate bool, args []string) []string {
	if !forceUpdate {
		return args
	}
	ret := args[:0]
	for _, arg := range args {
		if arg != "--force" {
			ret = append(ret, arg)
		}
	}
	return ret
}

// shouldUseXZ returns true if attempting to download the given version should use xzip compression.
func shouldUseXZ(version cli.Version) bool {
	return !version.LessThan(semver.Version{
		Major:      13,
		Minor:      2,
		PreRelease: "0", // Less than any valid prerelease string, e.g. alpha1
	})
}
