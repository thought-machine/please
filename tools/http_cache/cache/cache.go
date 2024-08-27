package cache

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/fs"
)

var log = logging.MustGetLogger("httpcache")

// Cache implements a http handler for caching files. Effectively a read/write http.FileSystem
type Cache struct {
	Dir string
}

// New create a new http cache
func New(dir string) *Cache {
	return &Cache{
		Dir: dir,
	}
}

// ServeHTTP implements the http.Handler interface for the cache
func (c *Cache) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	uri := req.RequestURI
	if req.Method == http.MethodPut {
		err := c.store(uri, req.Body)
		if err != nil {
			log.Errorf("Failed to store in cache: %v", err)
			resp.WriteHeader(http.StatusInternalServerError)
			_, _ = resp.Write([]byte(fmt.Sprintf("failed to store in cache: %v", err)))
		}
	} else if req.Method == http.MethodGet {
		http.ServeFile(resp, req, filepath.Join(c.Dir, uri))
	}
}

func (c *Cache) store(uri string, data io.Reader) error {
	path := filepath.Join(c.Dir, uri)
	if err := fs.RemoveAll(uri); err != nil {
		return err
	}

	if err := fs.EnsureDir(path); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, data)
	return err
}
