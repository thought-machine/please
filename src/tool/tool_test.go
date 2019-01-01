package tool

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

func TestMatchingTools(t *testing.T) {
	c, err := core.ReadConfigFiles(nil, "")
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"pex": "please_pex"}, matchingTools(c, "p"))
	assert.Equal(t, map[string]string{"pex": "please_pex"}, matchingTools(c, "pex"))
	assert.Equal(t, map[string]string{
		"javacworker": "javac_worker",
		"jarcat":      "jarcat",
	}, matchingTools(c, "ja"))
}

func TestAllToolNames(t *testing.T) {
	c, err := core.ReadConfigFiles(nil, "")
	assert.NoError(t, err)
	assert.Equal(t, []string{"jarcat", "javacworker"}, allToolNames(c, "ja"))
}
