package update

import (
	"archive/tar"
	"compress/bzip2"
	"io"
	"net/http"
	"os"
	"path"

	"core"
)

const url = "https://bitbucket.org/squeaky/portable-pypy/downloads/pypy-5.6-linux_x86_64-portable.tar.bz2"

// DownloadPyPy attempts to download a standalone PyPy distribution.
// We use this to try to deal with the case where there is no loadable interpreter.
// It also simplifies installation instructions on Linux where we can't use upstream packages
// often because they aren't built with --enable-shared.
// It returns the location of the downloaded libpypy-c.so
func DownloadPyPy(config *core.Configuration) (string, bool) {
	log.Notice("Checking if we've got a PyPy instance ready...")
	dest := core.ExpandHomePath(path.Join(config.Please.Location, "pypy"))
	so := path.Join(dest, "bin/libpypy-c.so")
	if core.PathExists(so) {
		log.Notice("Found PyPy at %s", so)
		return so, true
	}
	log.Warning("Attempting to download a portable PyPy distribution...")
	return so, downloadPyPy(dest)
}

func downloadPyPy(destination string) bool {
	if err := os.RemoveAll(destination); err != nil {
		log.Error("Can't remove %s: %s", destination, err)
		return false
	}
	resp, err := http.Get(url)
	if err != nil {
		log.Error("Failed to download PyPy: %s", err)
		return false
	}
	defer resp.Body.Close()
	bzreader := bzip2.NewReader(resp.Body)
	tarball := tar.NewReader(bzreader)
	for {
		hdr, err := tarball.Next()
		if err == io.EOF {
			break // End of archive
		} else if err != nil {
			log.Error("Error reading tarball: %s", err)
			return false
		} else if err := writeTarFile(hdr, tarball, destination); err != nil {
			log.Error("Error extracting tarball: %s", err)
			return false
		}
	}
	log.Notice("Downloaded PyPy successfully")
	return true
}
