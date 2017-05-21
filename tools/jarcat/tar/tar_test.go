package tar

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const testDataDir = "tools/jarcat/tar/test_data"

func TestNoCompression(t *testing.T) {
	filename := "test_no_compression.tar"
	err := Write(filename, testDataDir, "", false)
	assert.NoError(t, err)

	m := ReadTar(t, filename, false)
	assert.EqualValues(t, map[string]string{
		"dir1/file1.txt": "test file 1",
		"dir2/file2.txt": "test file 2",
	}, toFilenameMap(m))

	// All the timestamps should be fixated and there should be no user/group id.
	var zeroTime time.Time
	for h := range m {
		assert.EqualValues(t, mtime, h.ModTime.In(time.UTC))
		// These two seem to always be zero regardless of what we send in.
		// We don't really care as long as they're always the same.
		assert.EqualValues(t, zeroTime, h.AccessTime)
		assert.EqualValues(t, zeroTime, h.ChangeTime)
		assert.EqualValues(t, 0, h.Uid)
		assert.EqualValues(t, 0, h.Gid)
	}
}

func TestCompression(t *testing.T) {
	filename := "test_compression.tar.gz"
	err := Write(filename, testDataDir, "", true)
	assert.NoError(t, err)

	m := ReadTar(t, filename, true)
	assert.EqualValues(t, map[string]string{
		"dir1/file1.txt": "test file 1",
		"dir2/file2.txt": "test file 2",
	}, toFilenameMap(m))
}

func TestWithPrefix(t *testing.T) {
	filename := "test_prefix.tar"
	err := Write(filename, testDataDir, "wibble", false)
	assert.NoError(t, err)

	m := ReadTar(t, filename, false)
	assert.EqualValues(t, map[string]string{
		"wibble/file1.txt": "test file 1",
		"wibble/file2.txt": "test file 2",
	}, toFilenameMap(m))
}

// ReadTar is a test utility that reads all the files from a tarball and returns a map of
// their headers -> their contents.
func ReadTar(t *testing.T, filename string, compress bool) map[*tar.Header]string {
	f, err := os.Open(filename)
	assert.NoError(t, err)
	if compress {
		r, err := gzip.NewReader(f)
		assert.NoError(t, err)
		return readTar(t, r)
	}
	return readTar(t, f)
}

// readTar is a test utility that reads all the files from a tarball and returns a map of
// their headers -> their contents.
func readTar(t *testing.T, r io.Reader) map[*tar.Header]string {
	tr := tar.NewReader(r)
	m := map[*tar.Header]string{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		assert.NoError(t, err)
		var buf bytes.Buffer
		_, err = io.Copy(&buf, tr)
		assert.NoError(t, err)
		m[hdr] = strings.TrimSpace(buf.String()) // Don't worry about newline, they're just test files...
	}
	return m
}

// toFilenameMap converts one of the maps returned by above to a map of filenames to contents.
func toFilenameMap(m map[*tar.Header]string) map[string]string {
	r := map[string]string{}
	for hdr, contents := range m {
		r[hdr.Name] = contents
	}
	return r
}
