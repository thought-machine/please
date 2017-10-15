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

func TestAddFiles(t *testing.T) {
	f := NewFile("add_files_test.zip", false)
	f.Suffix = []string{"zip"}
	err := f.AddFiles("tools")
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
	err := f.AddFiles("tools")
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
