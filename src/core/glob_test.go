// Tests for our glob functions.

package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanGlobFirstFile(t *testing.T) {
	// If this fails then we probably failed to interpret /**/ properly,
	// which can resolve to just / - ie. we glob test_data/**/*.txt,
	// which should include test_data/test.txt
	if !FileExists("src/core/test_data/test.txt") {
		t.Errorf("Can't load test_data/test.txt")
	}
}

func TestCanGlobSecondFile(t *testing.T) {
	// If this fails then we haven't walked down enough subdirectories
	// or something. Shouldn't really be hard - it's a sanity check really
	// since it's similar to the third file but without a package boundary.
	if !FileExists("src/core/test_data/test_subfolder1/a.txt") {
		t.Errorf("Can't load test_data/test_subfolder1/a.txt")
	}
}

func TestCannotGlobThirdFile(t *testing.T) {
	// This one we should not be able to glob because it's inside its own subpackage.
	if FileExists("src/core/test_data/test_subfolder2/b.txt") {
		t.Errorf("Incorrectly loaded test_data/test_subfolder2/b.txt; have globbed it through a package boundary")
	}
}

func TestCanGlobFileAtRootWithDoubleStar(t *testing.T) {
	files, err := glob("src/core/test_data/test_subfolder1", "**/*.txt", false, nil)
	assert.NoError(t, err)
	assert.Equal(t, []string{"src/core/test_data/test_subfolder1/a.txt"}, files)
}

func TestIsGlob(t *testing.T) {
	assert.True(t, IsGlob("a*b"))
	assert.True(t, IsGlob("ab/*.txt"))
	assert.True(t, IsGlob("ab/c.tx?"))
	assert.True(t, IsGlob("ab/[a-z].txt"))
	assert.False(t, IsGlob("abc.txt"))
	assert.False(t, IsGlob("ab/c.txt"))
}

func TestGlobPlusPlus(t *testing.T) {
	files, err := glob("src/core/test_data/test_subfolder++", "**/*.txt", false, nil)
	assert.NoError(t, err)
	assert.Equal(t, []string{"src/core/test_data/test_subfolder++/test.txt"}, files)
}
