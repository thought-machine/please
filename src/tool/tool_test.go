package tool

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

func TestMatchingTools(t *testing.T) {
	c, err := core.ReadConfigFiles(os.DirFS("."), nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"langserver": "//_please:build_langserver"}, matchingTools(c, "la"))
	assert.Equal(t, map[string]string{"langserver": "//_please:build_langserver"}, matchingTools(c, "lang"))
	assert.Equal(t, map[string]string{
		"javacworker": "/////_please:javac_worker",
	}, matchingTools(c, "ja"))
}

func TestAllToolNames(t *testing.T) {
	c, err := core.ReadConfigFiles(os.DirFS("."), nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, []string{"javacworker"}, allToolNames(c, "ja"))
}
