package fs

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLink(t *testing.T) {
	var tests = []struct {
		description string
		srcExists   bool
		destExists  bool
		returnsErr  error
	}{
		{
			"src exists, dest does not exist",
			true, false, nil,
		},
		{
			"src exists, dest exists",
			true, true, nil,
		},
		{
			"src does not exist, dest exists",
			false, true, os.ErrNotExist,
		},
		{
			"src does not exist, dest does not exist",
			false, false, os.ErrNotExist,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			dir, err := os.MkdirTemp("", "testlink")
			require.NoError(t, err)
			defer os.RemoveAll(dir)

			src := path.Join(dir, "src")
			if tt.srcExists {
				require.NoError(t, os.WriteFile(src, []byte(tt.description+" src"), 0600))
			}
			dest := path.Join(dir, "dest")
			if tt.destExists {
				require.NoError(t, os.WriteFile(dest, []byte(tt.description+" dest"), 0600))
			}

			err = Link(src, dest)
			if tt.returnsErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// if there's no error, we expect the contents of the files to be the same.
				srcFileContents, err := os.ReadFile(src)
				require.NoError(t, err)
				destFileContents, err := os.ReadFile(dest)
				require.NoError(t, err)

				assert.Equal(t, string(srcFileContents), string(destFileContents))
			}
		})
	}
}

func TestSymlink(t *testing.T) {
	var tests = []struct {
		description string
		srcExists   bool
		destExists  bool
		returnsErr  error
	}{
		{
			"src exists, dest does not exist",
			true, false, nil,
		},
		{
			"src exists, dest exists",
			true, true, nil,
		},
		{
			"src does not exist, dest exists",
			false, true, os.ErrNotExist,
		},
		{
			"src does not exist, dest does not exist",
			false, false, os.ErrNotExist,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			dir, err := os.MkdirTemp("", "testlink")
			require.NoError(t, err)
			defer os.RemoveAll(dir)

			src := path.Join(dir, "src")
			if tt.srcExists {
				require.NoError(t, os.WriteFile(src, []byte(tt.description+" src"), 0600))
			}
			dest := path.Join(dir, "dest")
			if tt.destExists {
				require.NoError(t, os.WriteFile(dest, []byte(tt.description+" dest"), 0600))
			}

			err = Symlink(src, dest)
			if tt.returnsErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// if there's no error, we expect the contents of the files to be the same.
				srcFileContents, err := os.ReadFile(src)
				require.NoError(t, err)
				destFileContents, err := os.ReadFile(dest)
				require.NoError(t, err)

				assert.Equal(t, string(srcFileContents), string(destFileContents))
			}
		})
	}
}
