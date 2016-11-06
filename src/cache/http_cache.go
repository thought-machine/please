// Http-based cache.

package cache

import (
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"

	"core"
)

type httpCache struct {
	Url       string
	Writeable bool
	Timeout   time.Duration
	OSName    string
}

func (cache *httpCache) Store(target *core.BuildTarget, key []byte) {
	// TODO(pebers): Change this to upload using multipart, it's quite slow doing every file separately
	//               for targets with many files.
	if cache.Writeable {
		for out := range cacheArtifacts(target) {
			if info, err := os.Stat(out); err == nil && info.IsDir() {
				filepath.Walk(out, func(name string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					} else if !info.IsDir() {
						cache.StoreExtra(target, key, name)
					}
					return nil
				})
			} else {
				cache.StoreExtra(target, key, out)
			}
		}
	}
}

func (cache *httpCache) StoreExtra(target *core.BuildTarget, key []byte, file string) {
	if cache.Writeable {
		artifact := path.Join(
			cache.OSName,
			target.Label.PackageName,
			target.Label.Name,
			base64.RawURLEncoding.EncodeToString(key),
			file,
		)
		log.Info("Storing %s: %s in http cache...", target.Label, artifact)

		// NB. Don't need to close this file, http.Post will do it for us.
		file, err := os.Open(path.Join(target.OutDir(), file))
		if err != nil {
			log.Warning("Failed to read artifact: %s", err)
			return
		}
		response, err := http.Post(cache.Url+"/artifact/"+artifact, "application/octet-stream", file)
		if err != nil {
			log.Warning("Failed to send artifact to %s: %s", cache.Url+"/artifact/"+artifact, err)
		} else if response.StatusCode < 200 || response.StatusCode > 299 {
			log.Warning("Failed to send artifact to %s: got response %s", cache.Url+"/artifact/"+artifact, response.Status)
		}
		response.Body.Close()
	}
}

func (cache *httpCache) Retrieve(target *core.BuildTarget, key []byte) bool {
	// We can't tell from outside if this works or not (as we can for the dir cache)
	// so we must assume that a target with no artifacts can't be retrieved. It's a weird
	// case but a test already exists in the plz test suite so...
	retrieved := false
	for out := range cacheArtifacts(target) {
		if !cache.RetrieveExtra(target, key, out) {
			return false
		}
		retrieved = true
	}
	return retrieved
}

func (cache *httpCache) RetrieveExtra(target *core.BuildTarget, key []byte, file string) bool {
	log.Debug("Retrieving %s:%s from http cache...", target.Label, file)

	artifact := path.Join(
		cache.OSName,
		target.Label.PackageName,
		target.Label.Name,
		base64.RawURLEncoding.EncodeToString(key),
		file,
	)

	response, err := http.Get(cache.Url + "/artifact/" + artifact)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	if response.StatusCode == 404 {
		return false
	} else if response.StatusCode < 200 || response.StatusCode > 299 {
		log.Warning("Error %d from http cache", response.StatusCode)
		return false
	} else if response.Header.Get("Content-Type") == "application/octet-stream" {
		// Single artifact
		return cache.writeFile(target, file, response.Body)
	} else if _, params, err := mime.ParseMediaType(response.Header.Get("Content-Type")); err != nil {
		log.Warning("Couldn't parse response: %s", err)
		return false
	} else {
		// Directory, comes back in multipart
		mr := multipart.NewReader(response.Body, params["boundary"])
		for {
			if part, err := mr.NextPart(); err == io.EOF {
				return true
			} else if err != nil {
				log.Warning("Error reading multipart response: %s", err)
				return false
			} else if !cache.writeFile(target, part.FileName(), part) {
				return false
			}
		}
	}
}

func (cache *httpCache) writeFile(target *core.BuildTarget, file string, r io.Reader) bool {
	outFile := path.Join(target.OutDir(), file)
	if err := os.MkdirAll(path.Dir(outFile), core.DirPermissions); err != nil {
		log.Errorf("Failed to create directory: %s", err)
		return false
	}
	f, err := os.OpenFile(outFile, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, fileMode(target))
	if err != nil {
		log.Errorf("Failed to open file: %s", err)
		return false
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		log.Errorf("Failed to write file: %s", err)
		return false
	}
	log.Info("Retrieved %s from http cache", target.Label)
	return true
}

func (cache *httpCache) Clean(target *core.BuildTarget) {
	var reader io.Reader
	artifact := path.Join(
		cache.OSName,
		target.Label.PackageName,
		target.Label.Name,
	)
	req, _ := http.NewRequest("DELETE", cache.Url+"/artifact/"+artifact, reader)
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Warning("Failed to remove artifacts for %s from http cache: %s", target.Label, err)
	}
	response.Body.Close()
}

func (cache *httpCache) Shutdown() {}

func newHttpCache(config *core.Configuration) *httpCache {
	cache := new(httpCache)
	cache.OSName = runtime.GOOS + "_" + runtime.GOARCH
	cache.Url = config.Cache.HttpUrl
	cache.Writeable = config.Cache.HttpWriteable
	cache.Timeout = time.Duration(config.Cache.HttpTimeout)
	return cache
}
