package remote

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

const pleaseCacheDirName = "please"
const storeDirectoryName = "build-metadata-store"

type buildMetadataStore interface {
	storeMetadata(key string, md *core.BuildMetadata) error
	retrieveMetadata(key string) (*core.BuildMetadata, error)
}

type directoryMetadataStore struct {
	instance      string
	directory     string
	cacheDuration time.Duration
}

func newDirMDStore(instance string, cacheDuration time.Duration) *directoryMetadataStore {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Fatalf("failed to find user cache dir for metadata store: %v", err)
	}
	dir := filepath.Join(userCacheDir, pleaseCacheDirName, storeDirectoryName)

	if err := os.MkdirAll(dir, fs.DirPermissions); err != nil {
		log.Fatalf("failed to create metadata store directory: %v", err)
	}
	store := &directoryMetadataStore{
		instance:      instance,
		directory:     dir,
		cacheDuration: cacheDuration,
	}

	go store.clean()
	return store
}

// clean will delete any cached metadata that has expired
func (d *directoryMetadataStore) clean() {
	_ = fs.Walk(d.directory, func(name string, isDir bool) error {
		if isDir {
			return nil
		}

		if md, err := loadMetadata(name); err == nil && d.hasExpired(md) {
			_ = os.Remove(name)
		}
		return nil
	})
}

func (d *directoryMetadataStore) storeMetadata(key string, md *core.BuildMetadata) error {
	md.VersionTag = core.ExpectedBuildMetadataVersionTag

	prefix := key[:2]
	dir := filepath.Join(d.directory, d.instance, prefix)
	if err := os.MkdirAll(dir, fs.DirPermissions); err != nil {
		return fmt.Errorf("failed to create metadata store directory: %w", err)
	}

	filename := filepath.Join(dir, key)
	var buf bytes.Buffer
	writer := gob.NewEncoder(&buf)
	if err := writer.Encode(md); err != nil {
		return fmt.Errorf("failed to encode build metadata file: %w", err)
	}
	return fs.WriteFile(&buf, filename, 0644)
}

func (d *directoryMetadataStore) retrieveMetadata(key string) (*core.BuildMetadata, error) {
	prefix := key[:2]
	fileName := filepath.Join(d.directory, d.instance, prefix, key)

	md, err := loadMetadata(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	if d.hasExpired(md) {
		return nil, nil
	}
	// If the cached version metadata doesn't match the expected version, then we should not use this metadata file
	if md.VersionTag != core.ExpectedBuildMetadataVersionTag {
		return nil, nil
	}
	return md, nil
}

func (d *directoryMetadataStore) hasExpired(md *core.BuildMetadata) bool {
	return time.Since(md.Timestamp) > d.cacheDuration
}

func loadMetadata(fileName string) (*core.BuildMetadata, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	md := new(core.BuildMetadata)

	reader := gob.NewDecoder(file)
	if err := reader.Decode(&md); err != nil {
		return nil, err
	}
	return md, nil
}
