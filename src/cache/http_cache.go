// Http-based cache.

package cache

import (
	"archive/tar"
	"compress/gzip"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/hashicorp/go-retryablehttp"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/process"
)

type httpCache struct {
	url      string
	writable bool
	client   *retryablehttp.Client

	requestLimiter limiter
}

type limiter chan struct{}

func (l limiter) acquire() {
	l <- struct{}{}
}

func (l limiter) release() {
	<-l
}

// mtime is the time we attach for the modification time of all files.
var mtime = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)

// nobody is the usual uid / gid of the 'nobody' user.
const nobody = 65534

func (cache *httpCache) Store(target *core.BuildTarget, key []byte, files []string) {
	if cache.writable {
		cache.requestLimiter.acquire()
		defer cache.requestLimiter.release()

		r, w := io.Pipe()
		go cache.write(w, target, files)
		req, err := retryablehttp.NewRequest(http.MethodPut, cache.makeURL(key), r)
		if err != nil {
			log.Warning("Invalid cache URL: %s", err)
			return
		}
		if resp, err := cache.client.Do(req); err != nil {
			log.Warning("Failed to store files in HTTP cache: %s", err)
		} else {
			resp.Body.Close()
		}
	}
}

// makeURL returns the remote URL for a key.
func (cache *httpCache) makeURL(key []byte) string {
	return cache.url + "/" + hex.EncodeToString(key)
}

// write writes a series of files into the given Writer.
func (cache *httpCache) write(w io.WriteCloser, target *core.BuildTarget, files []string) {
	defer w.Close()
	gzw := gzip.NewWriter(w)
	defer gzw.Close()
	tw := tar.NewWriter(gzw)
	defer tw.Close()
	outDir := target.OutDir()

	for _, out := range files {
		if err := fs.Walk(path.Join(outDir, out), func(name string, isDir bool) error {
			return storeFile(tw, name)
		}); err != nil {
			log.Warning("Error uploading artifacts to HTTP cache: %s", err)
			// TODO(peterebden): How can we cancel the request at this point?
		}
	}
}

func storeFile(tw *tar.Writer, name string) error {
	info, err := os.Lstat(name)
	if err != nil {
		return err
	}
	target := ""
	if info.Mode()&os.ModeSymlink != 0 {
		target, _ = os.Readlink(name)
	}
	hdr, err := tar.FileInfoHeader(info, target)
	if err != nil {
		return err
	}
	hdr.Name = name
	// Zero out all timestamps.
	hdr.ModTime = mtime
	hdr.AccessTime = mtime
	hdr.ChangeTime = mtime
	// Strip user/group ids.
	hdr.Uid = nobody
	hdr.Gid = nobody
	hdr.Uname = "nobody"
	hdr.Gname = "nobody"
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	} else if info.IsDir() || target != "" {
		return nil // nothing to write
	}
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(tw, f)
	return err
}

func (cache *httpCache) Retrieve(target *core.BuildTarget, key []byte, _ []string) bool {
	cache.requestLimiter.acquire()
	defer cache.requestLimiter.release()

	m, err := cache.retrieve(key)
	if err != nil {
		log.Warning("%s: Failed to retrieve files from HTTP cache: %s", target.Label, err)
	}
	return m
}

func (cache *httpCache) retrieve(key []byte) (bool, error) {
	req, err := retryablehttp.NewRequest(http.MethodGet, cache.makeURL(key), nil)
	if err != nil {
		return false, err
	}
	resp, err := cache.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return false, nil // doesn't exist - not an error
	} else if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("%s", string(b))
	}
	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return false, err
	}
	defer gzr.Close()
	return readTar(gzr)
}

func readTar(gzr io.Reader) (bool, error) {
	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				return true, nil
			}
			return false, err
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(hdr.Name, core.DirPermissions); err != nil {
				return false, err
			}
		case tar.TypeReg:
			if dir := path.Dir(hdr.Name); dir != "." {
				if err := os.MkdirAll(dir, core.DirPermissions); err != nil {
					return false, err
				}
			}
			if f, err := openFile(hdr); err != nil {
				return false, err
			} else if _, err := io.Copy(f, tr); err != nil {
				return false, err
			} else if err := f.Close(); err != nil {
				return false, err
			}
		case tar.TypeSymlink:
			if err := os.Symlink(hdr.Linkname, hdr.Name); err != nil {
				return false, err
			}
		default:
			log.Warning("Unhandled file type %d for %s", hdr.Typeflag, hdr.Name)
		}
	}
}

func openFile(header *tar.Header) (*os.File, error) {
	f, err := os.OpenFile(header.Name, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, os.FileMode(header.Mode))
	if err != nil {
		if os.IsPermission(err) {
			// The file might already exist and be ro. If so, remove it.
			if err := fs.ForceRemove(process.New(), header.Name); err != nil {
				log.Debug("failed to remove existing file when restoring from the cache: %w", err)
			}
			return os.OpenFile(header.Name, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, os.FileMode(header.Mode))
		}
		return nil, err
	}
	return f, nil
}

func (cache *httpCache) Clean(*core.BuildTarget) {
	// Not possible; this implementation can only clean for a hash.
}

func (cache *httpCache) CleanAll() {
	// Also not possible.
}

func (cache *httpCache) Shutdown() {}

func newHTTPCache(config *core.Configuration) *httpCache {
	return &httpCache{
		url:      config.Cache.HTTPURL.String(),
		writable: config.Cache.HTTPWriteable,
		client: &retryablehttp.Client{
			HTTPClient: &http.Client{
				Timeout: time.Duration(config.Cache.HTTPTimeout),
			},
			Logger:       &cli.HTTPLogWrapper{Log: log},
			RetryWaitMin: 1 * time.Second,
			RetryWaitMax: 30 * time.Second,
			RetryMax:     config.Cache.HTTPRetry,
			CheckRetry:   retryablehttp.DefaultRetryPolicy,
			Backoff:      retryablehttp.DefaultBackoff,
		},
		requestLimiter: make(limiter, config.Cache.HTTPConcurrentRequestLimit),
	}
}
