package help

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPublicInterface(t *testing.T) {
	// Quick test on the main Help function; it doesn't expose much information so other
	// tests use non-exported functions to get a bit more detail.
	assert.True(t, Help("go_binary"))
	assert.False(t, Help("go_binaryz"))
	assert.False(t, Help("wibble"))
}

func TestHelpDescription(t *testing.T) {
	// The returned message should describe what kind of a thing it is
	assert.Contains(t, help("go_binary"), "built-in build rule")
	// And what its name was
	assert.Contains(t, help("go_binary"), "go_binary")
}

func TestSuggestion(t *testing.T) {
	assert.Equal(t, "\nMaybe you meant cc_embed_binary or c_embed_binary ?", suggest("cc_emdbed_binary"))
	assert.Equal(t, "", suggest("blahdiblahdiblah"))
}

func TestConfig(t *testing.T) {
	assert.Contains(t, help("NumThreads"), "config setting")
	assert.Contains(t, help("numthreads"), "config setting")
}

func TestMisc(t *testing.T) {
	assert.Contains(t, help("plzconfig"), "plzconfig")
}

func TestGeneralMessage(t *testing.T) {
	// Should provide some useful message for just "plz halp"
	assert.NotEqual(t, "", help(""))
}

func TestTopics(t *testing.T) {
	assert.NotEqual(t, "", help("topics"))
}
