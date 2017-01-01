package help

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPublicInterface(t *testing.T) {
	// Quick test on the main Help function; it doesn't expose much information so other
	// tests use non-exported functions to get a bit more detail.
	assert.True(t, Help("go_binary"))
	assert.False(t, Help("wibble"))
}

func TestHelpDescription(t *testing.T) {
	// The returned message should describe what kind of a thing it is
	assert.Contains(t, help("go_binary"), "built-in build rule")
	// And what its name was
	assert.Contains(t, help("go_binary"), "go_binary")
}
