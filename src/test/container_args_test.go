package test

import "os"
import "testing"

func TestContainerArgs(t *testing.T) {
	// This value will only be set if we passed the argument to Docker correctly.
	if env := os.Getenv("TEST_VALUE"); env != "WIBBLE" {
		t.Errorf("Incorrect TEST_VALUE env var; was %s", env)
	}
}
