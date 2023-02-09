package help

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

func TestMain(t *testing.M) {
	err := os.WriteFile(".plzconfig", nil, 0444)
	if err != nil {
		panic(err)
	}

	os.Exit(t.Run())
}

func TestPublicInterface(t *testing.T) {
	// Quick test on the main Help function; it doesn't expose much information so other
	// tests use non-exported functions to get a bit more detail.
	assert.True(t, Help("go_binary"))
	assert.False(t, Help("go_binaryz"))
	assert.False(t, Help("wibble"))
}

func TestHelpDescription(t *testing.T) {
	// The returned message should describe what kind of a thing it is
	assert.Contains(t, help("go_binary", core.DefaultConfiguration()), "built-in build rule")
	// And what its name was
	assert.Contains(t, help("go_binary", core.DefaultConfiguration()), "go_binary")
}

func TestSuggestion(t *testing.T) {
	assert.Equal(t, "\nMaybe you meant http_archive ?", suggest("http_archiv", core.DefaultConfiguration()))
	assert.Equal(t, "\nMaybe you meant godep or go ?", suggest("godop", core.DefaultConfiguration()))
	assert.Equal(t, "", suggest("blahdiblahdiblah", core.DefaultConfiguration()))
}

func TestConfig(t *testing.T) {
	assert.Contains(t, help("NumThreads", core.DefaultConfiguration()), "config setting")
	assert.Contains(t, help("numthreads", core.DefaultConfiguration()), "config setting")
}

func TestMisc(t *testing.T) {
	assert.Contains(t, help("plzconfig", core.DefaultConfiguration()), "plzconfig")
}

func TestGeneralMessage(t *testing.T) {
	// Should provide some useful message for just "plz halp"
	assert.NotEqual(t, "", help("", core.DefaultConfiguration()))
}

func TestTopics(t *testing.T) {
	assert.NotEqual(t, "", help("topics", core.DefaultConfiguration()))
}
