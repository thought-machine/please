// Http-based cache.

package cache

import (
	"encoding/base64"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

type httpCache struct {
	URL       string
	Writeable bool
	Timeout   time.Duration
}

func (cache *httpCache) Store(target *core.BuildTarget, key []byte, metadata *core.BuildMetadata, files []string) {
	// TODO(pebers): Change this to upload using multipart, it's quite slow doing every file separately
	//               for targets with many files.
	if cache.Writeable {
		for _, out := range files {
			if info, err := os.Stat(out); err == nil && info.IsDir() {
				fs.Walk(out, func(name string, isDir bool) error {
					if !isDir {
						cache.storeOne(target, key, name)
					}
					return nil
				})
			} else {
				cache.storeOne(target, key, out)
			}
		}
		if needsPostBuildFile(target) {
			cache.storeOne(target, key, target.PostBuildOutputFileName())
		}
	}
}

func (cache *httpCache) storeOne(target *core.BuildTarget, key []byte, file string) {
	if cache.Writeable {
		artifact := path.Join(
			core.OsArch,
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
		response, err := http.Post(cache.URL+"/artifact/"+artifact, "application/octet-stream", file)
		if err != nil {
			log.Warning("Failed to send artifact to %s: %s", cache.URL+"/artifact/"+artifact, err)
			return
		} else if response.StatusCode < 200 || response.StatusCode > 299 {
			log.Warning("Failed to send artifact to %s: got response %s", cache.URL+"/artifact/"+artifact, response.Status)
		}
		response.Body.Close()
	}
}

func (cache *httpCache) Retrieve(target *core.BuildTarget, key []byte) *core.BuildMetadata {
	// We can't tell from outside if this works or not (as we can for the dir cache)
	// so we must assume that a target with no artifacts can't be retrieved. It's a weird
	// case but a test already exists in the plz test suite so...
	var metadata *core.BuildMetadata
	for _, out := range target.Outputs() {
		if !cache.retrieveOne(target, key, out) {
			return nil
		}
		metadata = &core.BuildMetadata{}
	}
	if needsPostBuildFile(target) {
		if !cache.retrieveOne(target, key, target.PostBuildOutputFileName()) {
			return nil
		}
		metadata = loadPostBuildFile(target)
	}
	return metadata
}

func (cache *httpCache) retrieveOne(target *core.BuildTarget, key []byte, file string) bool {
	log.Debug("Retrieving %s:%s from http cache...", target.Label, file)

	artifact := path.Join(
		core.OsArch,
		target.Label.PackageName,
		target.Label.Name,
		base64.RawURLEncoding.EncodeToString(key),
		file,
	)

	response, err := http.Get(cache.URL + "/artifact/" + artifact)
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
	f, err := os.OpenFile(outFile, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, target.OutMode())
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
		core.OsArch,
		target.Label.PackageName,
		target.Label.Name,
	)
	req, _ := http.NewRequest("DELETE", cache.URL+"/artifact/"+artifact, reader)
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Warning("Failed to remove artifacts for %s from http cache: %s", target.Label, err)
	}
	response.Body.Close()
}

func (cache *httpCache) CleanAll() {
	req, _ := http.NewRequest("DELETE", cache.URL, nil)
	if _, err := http.DefaultClient.Do(req); err != nil {
		log.Warning("Failed to remove artifacts from http cache: %s", err)
	}
}

func (cache *httpCache) Shutdown() {}

func newHTTPCache(config *core.Configuration) *httpCache {
	return &httpCache{
		URL:       config.Cache.HTTPURL.String(),
		Writeable: config.Cache.HTTPWriteable,
		Timeout:   time.Duration(config.Cache.HTTPTimeout),
	}
}

// Convenience function to load a post-build output file after retrieving it from the cache.
func loadPostBuildFile(target *core.BuildTarget) *core.BuildMetadata {
	b, err := ioutil.ReadFile(path.Join(target.OutDir(), target.PostBuildOutputFileName()))
	if err != nil {
		return nil
	}
	return &core.BuildMetadata{Stdout: b}
}

// Another one to work out if we should try to store/retrieve a post-build output file.
func needsPostBuildFile(target *core.BuildTarget) bool {
	return target.PostBuildFunction != nil && target.State() < core.Built
}
