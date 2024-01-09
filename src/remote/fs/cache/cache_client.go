package cache

import (
	"context"
	"os"
	"path/filepath"

	"github.com/bazelbuild/remote-apis-sdks/go/pkg/client"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/digest"

	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/remote/fs"
)

var log = logging.Log

func New(client fs.Client, dir string) *Client {
	return &Client{
		dir:    dir,
		client: client,
	}
}

type Client struct {
	dir    string
	client fs.Client
}

func (c *Client) ReadBlob(ctx context.Context, d digest.Digest) ([]byte, *client.MovedBytesMetadata, error) {
	// A shortcut that allows us to use a len of 0 to mean a cache miss later on
	if d.Size == 0 {
		return nil, nil, nil
	}

	bs := c.read(d)
	if len(bs) != 0 {
		return bs, nil, nil
	}

	bs, md, err := c.client.ReadBlob(ctx, d)
	if err != nil {
		return nil, nil, err
	}
	if err := c.store(d, bs); err != nil {
		log.Warningf("failed to store blob in local CAS cache: %v", err)
	}
	return bs, md, nil
}

func (c *Client) read(d digest.Digest) []byte {
	path := c.pathForDigest(d)
	if _, err := os.Lstat(path); err != nil {
		return nil
	}
	bs, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return bs
}

func (c *Client) store(d digest.Digest, bs []byte) error {
	path := c.pathForDigest(d)
	if err := os.MkdirAll(filepath.Dir(path), os.ModeDir|0775); err != nil {
		return err
	}
	return os.WriteFile(path, bs, 0644)
}

func (c *Client) pathForDigest(d digest.Digest) string {
	return filepath.Join(c.dir, d.Hash[:2], d.Hash)
}
