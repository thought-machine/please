package plzinit

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thought-machine/please/src/fs"
)

const expectedRule = `
github_repo(
  name = "pleasings",
  repo = "thought-machine/pleasings",
  revision = "master",
)
`

func TestInitPleasings(t *testing.T) {
	err := InitPleasings("BUILD", true, "master")
	require.NoError(t, err)

	assert.False(t, fs.FileExists("BUILD"))

	err = InitPleasings("BUILD", false, "master")
	require.NoError(t, err)

	b, err := os.ReadFile("BUILD")
	require.NoError(t, err)

	assert.Equal(t, expectedRule, string(b))
}
