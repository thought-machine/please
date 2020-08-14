package cache

import (
	"fmt"
	"github.com/thought-machine/please/src/fs"
	"gopkg.in/op/go-logging.v1"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

var log = logging.MustGetLogger("httpcache")


type Cache struct {
	Dir string
}

func New(dir string) *Cache {
	return &Cache{
		Dir: dir,
	}
}

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
	if err := os.RemoveAll(uri); err != nil {
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