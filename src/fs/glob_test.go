// Tests for our glob functions.

package fs

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var buildFileNames = []string{"TEST_BUILD", "BUILD"}

func glob(rootPath string, glob string, excludes []string, includeHidden bool) ([]string, error) {
	return NewGlobber(HostFS, buildFileNames).glob(rootPath, glob, excludes, includeHidden, true)
}

func TestCanGlobFileAtRootWithDoubleStar(t *testing.T) {
	files, err := glob("src/fs/test_data/test_subfolder1", "**/*.txt", nil, false)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"src/fs/test_data/test_subfolder1/a.txt",
		"src/fs/test_data/test_subfolder1/sub_sub_folder/b.txt",
	}, files)
}

func TestCanGlobDirectories(t *testing.T) {
	files, err := glob("src/fs", "test_data/*", nil, false)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"src/fs/test_data/test_subfolder++",
		"src/fs/test_data/test_subfolder1",
		"src/fs/test_data/test_subfolder3",
		"src/fs/test_data/test.txt",
	}, files)
}

func TestIsGlob(t *testing.T) {
	assert.True(t, IsGlob("a*b"))
	assert.True(t, IsGlob("ab/*.txt"))
	assert.True(t, IsGlob("ab/c.tx?"))
	assert.True(t, IsGlob("ab/[a-z].txt"))
	assert.False(t, IsGlob("abc.txt"))
	assert.False(t, IsGlob("ab/c.txt"))
}

func TestGlobRanges(t *testing.T) {
	files, err := glob("src/fs/test_data", "test_subfolder3/[a-z]est.py", nil, false)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"src/fs/test_data/test_subfolder3/test.py",
		"src/fs/test_data/test_subfolder3/best.py",
	}, files)
}

func TestGlobQuestion(t *testing.T) {
	files, err := glob("src/fs/test_data", "test_subfolder3/?est.py", nil, false)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"src/fs/test_data/test_subfolder3/test.py",
		"src/fs/test_data/test_subfolder3/best.py",
		"src/fs/test_data/test_subfolder3/Zest.py",
	}, files)
}

func TestGlobPlusPlusInDirName(t *testing.T) {
	files, err := glob("src/fs/test_data/test_subfolder++", "**/*.txt", nil, false)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"src/fs/test_data/test_subfolder++/test.txt"}, files)
}

func TestGlobExcludes(t *testing.T) {
	t.Run("relative glob", func(t *testing.T) {
		files := Glob(HostFS, buildFileNames, "src/fs", []string{"test_data/**.txt"}, []string{"test.txt"}, false)
		expected := []string{
			"test_data/test_subfolder1/a.txt",
			"test_data/test_subfolder1/sub_sub_folder/b.txt",
		}
		assert.ElementsMatch(t, expected, files)
	})

	t.Run("exact glob", func(t *testing.T) {
		files := Glob(HostFS, buildFileNames, "src/fs", []string{"test_data/**.txt"}, []string{"test_data/*.txt"}, false)
		expected := []string{
			"test_data/test_subfolder1/a.txt",
			"test_data/test_subfolder1/sub_sub_folder/b.txt",
			"test_data/test_subfolder++/test.txt",
		}
		assert.ElementsMatch(t, expected, files)
	})

	t.Run("entire directory", func(t *testing.T) {
		files := Glob(HostFS, buildFileNames, "src/fs", []string{"test_data/**.txt"}, []string{"test_data/test_subfolder1/**"}, false)
		expected := []string{
			"test_data/test_subfolder++/test.txt",
			"test_data/test.txt",
		}
		assert.ElementsMatch(t, expected, files)
	})

	t.Run("entire directory via base path exclusion", func(t *testing.T) {
		files := Glob(HostFS, buildFileNames, "src/fs", []string{"test_data/**.txt"}, []string{"test_data/test_subfolder1"}, false)
		expected := []string{
			"test_data/test_subfolder++/test.txt",
			"test_data/test.txt",
		}
		assert.ElementsMatch(t, expected, files)
	})

	t.Run("mix of relative and total", func(t *testing.T) {
		files := Glob(HostFS, buildFileNames, "src/fs", []string{"test_data/**.txt"}, []string{"test.txt", "test_data/test_subfolder1/a.txt"}, false)
		expected := []string{
			"test_data/test_subfolder1/sub_sub_folder/b.txt",
		}
		assert.ElementsMatch(t, expected, files)
	})
}

func TestCannotGlobBetweenPackageBoundaries(t *testing.T) {
	files := Glob(HostFS, buildFileNames, "src/fs", []string{"**/*.txt", "**/*.py"}, nil, false)
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

func TestCompilePattern(t *testing.T) {
	rootPath := "/folder"

	testCases := []struct {
		name          string
		pattern       string
		shouldMatch   string
		shouldntMatch string
	}{
		{
			name:          "exact match",
			pattern:       "test.txt",
			shouldMatch:   "/folder/test.txt",
			shouldntMatch: "/folder/test.py",
		},
		{
			name:          "range match",
			pattern:       "[a-z]est.txt",
			shouldMatch:   "/folder/test.txt",
			shouldntMatch: "/folder/Test.txt",
		},
		{
			name:          "wild card match",
			pattern:       "?est.txt",
			shouldMatch:   "/folder/test.txt",
			shouldntMatch: "/folder/teest.txt",
		},
		{
			name:          "astrix matches one char",
			pattern:       "*est.txt",
			shouldMatch:   "/folder/test.txt",
			shouldntMatch: "/folder/test.py",
		},
		{
			name:          "astrix matches many char",
			pattern:       "*.txt",
			shouldMatch:   "/folder/test.txt",
			shouldntMatch: "/folder/test.py",
		},
		{
			name:          "double astrix matches same dir",
			pattern:       "**.txt",
			shouldMatch:   "/folder/test.txt",
			shouldntMatch: "/folder/test.py",
		},
		{
			name:          "double astrix matches sub dir",
			pattern:       "**.txt",
			shouldMatch:   "/folder/a/test.txt",
			shouldntMatch: "/folder/a/test.py",
		},
		{
			name:          "**/*.txt matches sub dir",
			pattern:       "**/*.txt",
			shouldMatch:   "/folder/a/test.txt",
			shouldntMatch: "/folder/test.test",
		},
		{
			name:          "+ matched literally",
			pattern:       "a/a+.txt",
			shouldMatch:   "/folder/a/a+.txt",
			shouldntMatch: "/folder/a/aa.txt",
		},
		{
			name:          "*thing* matches anything with that in file name",
			pattern:       "a/*.thing*",
			shouldMatch:   "/folder/a/some.thing.txt",
			shouldntMatch: "/folder/a/aa.txt",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			p, err := patternToMatcher(rootPath, testCase.pattern)
			require.NoError(t, err)

			match, err := p.Match(testCase.shouldMatch)
			require.NoError(t, err)
			assert.True(t, match, fmt.Sprintf("%s should match %s", testCase.pattern, testCase.shouldMatch))

			match, err = p.Match(testCase.shouldntMatch)
			require.NoError(t, err)
			assert.False(t, match, fmt.Sprintf("%s shouldn't match %s", testCase.pattern, testCase.shouldntMatch))
		})
	}
}

func TestShouldExcludeMatch(t *testing.T) {
	testCases := []struct {
		testName      string
		matchName     string
		exclude       string
		shouldExclude bool
	}{
		{
			testName:      "relative match",
			matchName:     "src/test_data/test.txt",
			exclude:       "test.txt",
			shouldExclude: true,
		},
		{
			testName:      "exact match",
			matchName:     "src/test_data/test.txt",
			exclude:       "test_data/test.txt",
			shouldExclude: true,
		},
		{
			testName:      "relative no match",
			matchName:     "src/test_data/test.txt",
			exclude:       "test.py",
			shouldExclude: false,
		},
		{
			testName:      "exact no match",
			matchName:     "src/test_data/test.txt",
			exclude:       "test_data/test.py",
			shouldExclude: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			shouldExclude, err := shouldExcludeMatch("src/", testCase.matchName, []string{testCase.exclude})
			assert.NoError(t, err)
			assert.Equal(t, testCase.shouldExclude, shouldExclude)
		})
	}
}

func TestIsInDirectories(t *testing.T) {
	assert.False(t, isInDirectories("test.go", []string{"test"}))
	assert.False(t, isInDirectories("foo/test.go", []string{"test"}))
	assert.False(t, isInDirectories("testfoo/test.go", []string{"test"}))
	assert.True(t, isInDirectories("test/test.go", []string{"test"}))
	assert.True(t, isInDirectories("test/foo", []string{"test"}))
}
