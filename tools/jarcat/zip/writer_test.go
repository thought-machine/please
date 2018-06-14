package zip

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var expectedModTime = time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC)

func TestAddZipFile(t *testing.T) {
	// Have to write an actual file for zip.OpenReader to use later.
	f := NewFile("add_zip_file_test.zip", false)
	err := f.AddZipFile("tools/jarcat/zip/test_data/test.zip")
	require.NoError(t, err)
	f.Close()
	assertExpected(t, "add_zip_file_test.zip", 0)
}

// getZipContents is a helper which returns a map of filename -> contents
func getZipContents(zipfilePath string) (map[string][]byte, error) {
	reader, err := zip.OpenReader(zipfilePath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	res := make(map[string][]byte, len(reader.File))
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		fReader, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer fReader.Close()
		contents := make([]byte, file.FileInfo().Size())
		if _, err := io.ReadFull(fReader, contents); err != nil {
			return nil, err
		}
		res[file.FileInfo().Name()] = contents
	}
	return res, nil
}

func TestAddZipFileConcatenatesSpecialFiles(t *testing.T) {
	t.Run("Scala Akka reference.conf", func(t *testing.T) {
		r := require.New(t)
		f := NewFile("zip_files_with_reference_conf.zip", false)

		err := f.AddZipFile("tools/jarcat/zip/test_data_3/z1.zip")
		r.NoError(err)
		err = f.AddZipFile("tools/jarcat/zip/test_data_3/z2.zip")
		r.NoError(err)
		f.Close()

		actualContents, err := getZipContents("zip_files_with_reference_conf.zip")
		r.NoError(err)
		z1Contents, err := getZipContents("tools/jarcat/zip/test_data_3/z1.zip")
		r.NoError(err)
		z2Contents, err := getZipContents("tools/jarcat/zip/test_data_3/z2.zip")
		r.NoError(err)
		expectedRefConf := append(z1Contents["reference.conf"], z2Contents["reference.conf"]...)
		r.EqualValues(actualContents["reference.conf"], expectedRefConf)
		// OTOH, z2's file1 should just be ignored because it's not a special
		// case and z1 already added it
		expectedNormalFile := z1Contents["file1"]
		r.EqualValues(actualContents["file1"], expectedNormalFile)
	})
}

func TestAddFiles(t *testing.T) {
	f := NewFile("add_files_test.zip", false)
	f.Suffix = []string{"zip"}
	err := f.AddFiles("tools/jarcat/zip/test_data")
	require.NoError(t, err)
	err = f.AddFiles("tools/jarcat/zip/test_data_2")
	require.NoError(t, err)
	f.Close()
	assertExpected(t, "add_files_test.zip", 0)
}

func assertExpected(t *testing.T, filename string, alignment int) {
	r, err := zip.OpenReader(filename)
	require.NoError(t, err)
	defer r.Close()
	files := []struct{ Name, Prefix string }{
		{"build_step.go", "// Implementation of Step interface."},
		{"incrementality.go", "// Utilities to help with incremental builds."},
	}
	for i, f := range r.File {
		assert.Equal(t, f.Name, files[i].Name)
		assert.Equal(t, expectedModTime, f.ModTime())

		fr, err := f.Open()
		require.NoError(t, err)
		var buf bytes.Buffer
		_, err = io.Copy(&buf, fr)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(buf.String(), files[i].Prefix))
		fr.Close()

		if alignment > 0 {
			offset, err := f.DataOffset()
			assert.NoError(t, err)
			assert.True(t, int(offset)%alignment == 0)
		}
	}
}

func TestAlignment(t *testing.T) {
	for _, align := range []int{2, 4, 8, 12, 32} {
		t.Run(fmt.Sprintf("%dByte", align), func(t *testing.T) {
			filename := fmt.Sprintf("test_alignment_%d.zip", align)
			f := NewFile(filename, false)
			f.Align = align
			err := f.AddFiles("tools/jarcat/zip/test_data_2")
			require.NoError(t, err)
			f.Close()
			assertExpected(t, filename, align)
		})
	}
}

func TestStoreSuffix(t *testing.T) {
	// This is a sort of Android-esque example (storing PNGs at 4-byte alignment)
	f := NewFile("test_store_suffix.zip", false)
	f.Suffix = []string{"zip"}
	f.StoreSuffix = []string{"png"}
	f.Align = 4
	f.IncludeOther = true
	err := f.AddFiles("tools/jarcat/zip/test_data")
	require.NoError(t, err)
	err = f.AddFiles("tools/jarcat/zip/test_data_2")
	require.NoError(t, err)
	f.Close()

	r, err := zip.OpenReader("test_store_suffix.zip")
	require.NoError(t, err)
	defer r.Close()
	assert.Equal(t, 3, len(r.File))
	png := r.File[0]
	assert.Equal(t, "tools/jarcat/zip/test_data/kitten.png", png.Name)
	assert.Equal(t, zip.Store, png.Method)
}

func TestIsSamePath(t *testing.T) {
	assert.True(t, samePaths("a", "a"))
	assert.True(t, samePaths("a", "./a"))
	assert.True(t, samePaths("/a", "/a"))
	assert.False(t, samePaths("/a", "./a"))
}

func TestIsPy37(t *testing.T) {
	f := NewFile("test_is_py37.zip", false)
	assert.False(t, f.isPy37([]byte("\x03\xf3\r\n"))) // 2.7.15
	assert.False(t, f.isPy37([]byte("\xee\x0c\r\n"))) // 3.4.3
	assert.False(t, f.isPy37([]byte("3\r\r\n")))      // 3.6.2
	assert.True(t, f.isPy37([]byte("B\r\r\n")))       // 3.7rc1
	assert.False(t, f.isPy37([]byte{0, 0, 0, 0}))
	assert.False(t, f.isPy37([]byte{255, 255, 255, 255}))
}
