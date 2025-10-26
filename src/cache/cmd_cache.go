package cache

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/hex"
	"io"
	"os"
	"os/exec"
	"path/filepath"

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
		strKey := keyToString(key)
		log.Debug("Storing %s: %s in custom cache...", target.Label, strKey)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		cmd := exec.CommandContext(ctx, "sh", "-c", cache.storeCommand)
		cmd.Env = append(cmd.Env, "CACHE_KEY="+strKey)

		r, w := io.Pipe()
		cmd.Stdin = r

		go write(w, target, files, cancel)
		output, err := cmd.CombinedOutput()

		if err != nil {
			log.Warning("Failed to store files via custom command: %s", err)
			if len(output) > 0 {
				log.Warning("Custom command output:%s", string(output))
			}
		}
	}
}

func (cache *cmdCache) Retrieve(target *core.BuildTarget, key []byte, _ []string) bool {
	strKey := keyToString(key)
	log.Debug("Retrieve %s: %s from custom cache...", target.Label, strKey)

	cmd := exec.Command("sh", "-c", cache.retrieveCommand)
	cmd.Env = append(os.Environ(), "CACHE_KEY="+strKey)

	var cmdOutputBuffer bytes.Buffer
	cmd.Stderr = &cmdOutputBuffer

	r, w := io.Pipe()
	cmd.Stdout = w

	if err := cmd.Start(); err != nil {
		log.Warning("Unable to start custom retrieve command: %s", err)
		return false
	}

	cmdResult := make(chan bool)

	go func() {
		var ok bool

		if err := cmd.Wait(); err != nil {
			log.Warning("Unable to unpack tar from custom command: %s", err)
			if cmdOutputBuffer.Len() > 0 {
				log.Warning("Custom command output:%s", cmdOutputBuffer.String())
			}
			ok = false
		} else {
			if cmdOutputBuffer.Len() > 0 {
				log.Debug("Custom command output:%s", cmdOutputBuffer.String())
			}
			ok = true
		}

		// have to explicitly close the read here to potentially interrupt
		// a forever blocking tar reader in case that the command died
		// before even getting the first entry
		r.Close()

		cmdResult <- ok
	}()

	tarOk, err := readTar(r)
	if err != nil {
		log.Debug("Error in tar reader: %s", err)
	}

	return tarOk && <-cmdResult
}

func (cache *cmdCache) Clean(*core.BuildTarget) {
}

func (cache *cmdCache) CleanAll() {
}

func (cache *cmdCache) Shutdown() {
}

// write writes a series of files into the given Writer.
func write(w io.WriteCloser, target *core.BuildTarget, files []string, cancel context.CancelFunc) {
	defer w.Close()
	tw := tar.NewWriter(w)

	defer tw.Close()
	outDir := target.OutDir()

	for _, out := range files {
		if err := fs.Walk(filepath.Join(outDir, out), func(name string, isDir bool) error {
			return storeFile(tw, name)
		}); err != nil {
			log.Warning("Error sending artifacts to command-driven cache: %s", err)
			// kill the running command
			cancel()
			return
		}
	}
}

func newCmdCache(config *core.Configuration) *cmdCache {
	return &cmdCache{
		storeCommand:    config.Cache.StoreCommand,
		retrieveCommand: config.Cache.RetrieveCommand,
	}
}
