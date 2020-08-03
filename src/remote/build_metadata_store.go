package remote

import (
	"encoding/gob"
	"fmt"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"os"
	"path/filepath"
)

const pleaseCacheDirName = "please"
const storeDirectoryName = "build-metadata-store"

type buildMetadataStore interface {
	StoreMetadata(key string, md *core.BuildMetadata) error
	RetrieveMetadata(key string) (*core.BuildMetadata, error)
}

func newDirMDStore() *directoryMetadataStore {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		panic(fmt.Errorf("failed to find user cache dir for metadata store: %v", err))
	}
	dir := filepath.Join(userCacheDir, pleaseCacheDirName, storeDirectoryName)

	if err := os.MkdirAll(dir, fs.DirPermissions); err != nil {
		panic(fmt.Errorf("failed to create metadata store directory: %v", err))
	}
	return &directoryMetadataStore{
		directory: dir,
	}
}

type directoryMetadataStore struct {
	directory string
}

func (d *directoryMetadataStore) StoreMetadata(key string, md *core.BuildMetadata) error {
	filename := filepath.Join(d.directory, key)
	if err := os.RemoveAll(filename); err != nil {
		return err
	}

	mdFile, err := os.Create(filename)
	if err != nil {
		return err
	}

	defer mdFile.Close()

	writer := gob.NewEncoder(mdFile)
	if err := writer.Encode(md); err != nil {
		return fmt.Errorf("failed to encode build metadata file: %w", err)
	}
	return nil
}

func (d *directoryMetadataStore) RetrieveMetadata(key string) (*core.BuildMetadata, error) {
	file, err := os.Open(filepath.Join(d.directory, key))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
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
