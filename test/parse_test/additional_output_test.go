// Test on adding extra output files to a target in a post-build function.
package parse

import (
	"os"
	"testing"
)

func TestContentsOfOutputFile(t *testing.T) {
	contents, err := os.ReadFile("test/parse_test/test_additional_output.txt")
	if err != nil {
		t.Errorf("Failed to read additional output file: %s", err)
	}
	if string(contents) != "kittens" {
		t.Errorf("Unexpected file contents: was '%s', expected 'kittens'", string(contents))
	}
}
