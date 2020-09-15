// These tests use glob in the build rule to load data in. These check that it really doesn't allow us to glob across
// package boundaries
package fs

import (
	"os"
	"testing"
)

func TestCanGlobFileInRoot(t *testing.T) {
	// If this fails then we probably failed to interpret /**/ properly,
	// which can resolve to just / - ie. we glob test_data/**/*.txt,
	// which should include test_data/test.txt
	if _, err := os.Lstat("test_data/test.txt"); err != nil {
		t.Errorf("Can't load test_data/test.txt")
	}
}

func TestCanGlobFileInSubDirectory(t *testing.T) {
	// If this fails then we haven't walked down enough subdirectories
	// or something. Shouldn't really be hard - it's a sanity check really
	// since it's similar to the third file but without a package boundary.
	if _, err := os.Lstat("test_data/test_subfolder1/a.txt"); err != nil {
		t.Errorf("Can't load test_data/test_subfolder1/a.txt")
	}
}

func TestCannotGlobFileInSubPackage(t *testing.T) {
	// This one we should not be able to glob because it's inside its own subpackage.
	if _, err := os.Lstat("test_data/test_subfolder2/b.txt"); err == nil {
		t.Errorf("Incorrectly loaded test_data/test_subfolder2/b.txt; have globbed it through a package boundary")
	}
}
