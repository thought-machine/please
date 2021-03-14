// Tests for our glob functions.

package fs

import (
	fsglob "github.com/thought-machine/please/src/fs/glob"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var buildFileNames = []string{"TEST_BUILD", "BUILD"}

func glob(rootPath string, glob string, excludes []string, includeHidden bool) ([]string, error) {
	return fsglob.New(buildFileNames).Glob(rootPath, includeHidden, []string{glob}, excludes)
}

func TestCanGlobFileAtRootWithDoubleStar(t *testing.T) {
	files, err := glob("src/fs/test_data/test_subfolder1", "**/*.txt", nil, false)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"a.txt",
		"sub_sub_folder/b.txt",
	}, files)
}

func TestCanGlobDirectories(t *testing.T) {
	files, err := glob(".", "src/fs/test_data/*", nil, false)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"src/fs/test_data/test_subfolder++",
		"src/fs/test_data/test_subfolder1",
		"src/fs/test_data/test_subfolder3",
		"src/fs/test_data/test.txt",
	}, files)
}

func TestGlobRanges(t *testing.T) {
	files, err := glob("src/fs/test_data", "test_subfolder3/[a-z]est.py", nil, false)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"test_subfolder3/test.py",
		"test_subfolder3/best.py",
	}, files)
}

func TestGlobQuestion(t *testing.T) {
	files, err := glob("src/fs/test_data", "test_subfolder3/?est.py", nil, false)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"test_subfolder3/test.py",
		"test_subfolder3/best.py",
		"test_subfolder3/Zest.py",
	}, files)
}

func TestGlobPlusPlusInDirName(t *testing.T) {
	files, err := glob("src/fs/test_data/test_subfolder++", "**/*.txt", nil, false)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"test.txt"}, files)
}

func TestGlobExcludes(t *testing.T) {
	t.Run("relative glob", func(t *testing.T) {
		files := Glob(buildFileNames, "src/fs", []string{"test_data/**.txt"}, []string{"test.txt"}, false)
		expected := []string{
			"test_data/test_subfolder1/a.txt",
			"test_data/test_subfolder1/sub_sub_folder/b.txt",
		}
		assert.ElementsMatch(t, expected, files)
	})

	t.Run("exact glob", func(t *testing.T) {
		files := Glob(buildFileNames, "src/fs", []string{"test_data/**.txt"}, []string{"test_data/*.txt"}, false)
		expected := []string{
			"test_data/test_subfolder1/a.txt",
			"test_data/test_subfolder1/sub_sub_folder/b.txt",
			"test_data/test_subfolder++/test.txt",
		}
		assert.ElementsMatch(t, expected, files)
	})

	t.Run("entire directory", func(t *testing.T) {
		files := Glob(buildFileNames, "src/fs", []string{"test_data/**.txt"}, []string{"test_data/test_subfolder1/**"}, false)
		expected := []string{
			"test_data/test_subfolder++/test.txt",
			"test_data/test.txt",
		}
		assert.ElementsMatch(t, expected, files)
	})

	t.Run("entire directory via base path exclusion", func(t *testing.T) {
		files := Glob(buildFileNames, "src/fs", []string{"test_data/**.txt"}, []string{"test_data/test_subfolder1"}, false)
		expected := []string{
			"test_data/test_subfolder++/test.txt",
			"test_data/test.txt",
		}
		assert.ElementsMatch(t, expected, files)
	})

	t.Run("mix of relative and total", func(t *testing.T) {
		files := Glob(buildFileNames, "src/fs", []string{"test_data/**.txt"}, []string{"test.txt", "test_data/test_subfolder1/a.txt"}, false)
		expected := []string{
			"test_data/test_subfolder1/sub_sub_folder/b.txt",
		}
		assert.ElementsMatch(t, expected, files)
	})
}

func TestCannotGlobBetweenPackageBoundaries(t *testing.T) {
	files := Glob(buildFileNames, "src/fs", []string{"**/*.txt", "**/*.py"}, nil, false)
	expected := []string{
		"test_data/test_subfolder++/test.txt",
		"test_data/test_subfolder1/a.txt",
		"test_data/test_subfolder1/sub_sub_folder/b.txt",
		"test_data/test_subfolder3/test.py",
		"test_data/test_subfolder3/best.py",
		"test_data/test_subfolder3/Zest.py",
		"test_data/test.txt",
	}
	assert.ElementsMatch(t, expected, files)
}
