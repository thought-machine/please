// Tests for our glob functions.

package parse

import "testing"

import "core"

func TestCanGlobFirstFile(t *testing.T) {
	// If this fails then we probably failed to interpret /**/ properly,
	// which can resolve to just / - ie. we glob test_data/**/*.txt,
	// which should include test_data/test.txt
	if !core.FileExists("src/parse/test_data/test.txt") {
		t.Errorf("Can't load test_data/test.txt")
	}
}

func TestCanGlobSecondFile(t *testing.T) {
	// If this fails then we haven't walked down enough subdirectories
	// or something. Shouldn't really be hard - it's a sanity check really
	// since it's similar to the third file but without a package boundary.
	if !core.FileExists("src/parse/test_data/test_subfolder1/a.txt") {
		t.Errorf("Can't load test_data/test_subfolder1/a.txt")
	}
}

func TestCannotGlobThirdFile(t *testing.T) {
	// This one we should not be able to glob because it's inside its own subpackage.
	if core.FileExists("src/parse/test_data/test_subfolder2/b.txt") {
		t.Errorf("Incorrectly loaded test_data/test_subfolder2/b.txt; have globbed it through a package boundary")
	}
}
