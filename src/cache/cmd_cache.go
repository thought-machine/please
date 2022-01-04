package cache

import (
	"archive/tar"
	"bytes"
	"encoding/hex"
	"io"
	"os/exec"
	"path"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

type cmdCache struct {
	storeCommand    string
	retrieveCommand string
}

func keyToString(key []byte) string {
	return hex.EncodeToString(key)
}

func (cache *cmdCache) Store(target *core.BuildTarget, key []byte, files []string) {
	if cache.storeCommand != "" {
		cmd := exec.Command("sh", "-c", cache.storeCommand)
		cmd.Env = append(cmd.Env, "CACHE_KEY="+keyToString(key))

		r, w := io.Pipe()
		cmd.Stdin = r

		go cache.write(w, target, files)
		if output, err := cmd.CombinedOutput(); err != nil {
			log.Warning("Failed to store files via custom command: %s", err)
			log.Warning("Output was: %s", string(output))
		} else if len(output) > 0 {
			log.Info("Custom command output:%s", string(output))
		}
	}
}

func (cache *cmdCache) Retrieve(target *core.BuildTarget, key []byte, _ []string) bool {

	cmd := exec.Command("sh", "-c", cache.retrieveCommand)
	cmd.Env = append(cmd.Env, "CACHE_KEY="+keyToString(key))

	var output bytes.Buffer
	cmd.Stderr = &output

	r, w := io.Pipe()
	cmd.Stdout = w

	if err := cmd.Start(); err != nil {
		log.Warning("Unable to start custom retrieve command: %s", err)
		return false
	}

	ok, err := readTar(r)
	if err != nil {
		log.Warning("Unable to unpack tar from custom command: %s", err)
		return false
	}

	if err = cmd.Wait(); err != nil {
		log.Warning("Unable to unpack tar from custom command: %s", err)
		log.Warning("Output was: %s", string(output.Bytes()))
		return false
	} else if output.Len() > 0 {
		log.Info("Custom command output:%s", string(output.Bytes()))
	}

	return ok
}

func (cache *cmdCache) Clean(*core.BuildTarget) {
}

func (cache *cmdCache) CleanAll() {
}

func (cache *cmdCache) Shutdown() {
}

// write writes a series of files into the given Writer.
func (cache *cmdCache) write(w io.WriteCloser, target *core.BuildTarget, files []string) {
	defer w.Close()
	tw := tar.NewWriter(w)
	defer tw.Close()
	outDir := target.OutDir()

	for _, out := range files {
		if err := fs.Walk(path.Join(outDir, out), func(name string, isDir bool) error {
			return storeFile(tw, name)
		}); err != nil {
			log.Warning("Error sending artifacts to command-driven cache: %s", err)
		}
	}
}

func newCmdCache(config *core.Configuration) *cmdCache {
	return &cmdCache{
		storeCommand:    config.Cache.StoreCommand,
		retrieveCommand: config.Cache.RetrieveCommand,
	}
}
