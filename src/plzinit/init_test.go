package plzinit

import (
	"os"
	"path/filepath"
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

const expectedGoModFilegroup = `filegroup(
    name = "gomod",
    srcs = ["go.mod"],
    visibility = ["PUBLIC"],
)
`

func TestInitGoMod(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example_module\n"), 0644))

	label, err := initGoMod(dir)
	require.NoError(t, err)
	assert.Equal(t, "//:gomod", label)

	b, err := os.ReadFile(filepath.Join(dir, "BUILD"))
	require.NoError(t, err)
	assert.Equal(t, expectedGoModFilegroup, string(b))
}

func TestInitGoModWithoutGoMod(t *testing.T) {
	dir := t.TempDir()

	label, err := initGoMod(dir)
	require.NoError(t, err)
	assert.Equal(t, "", label)
	assert.False(t, fs.FileExists(filepath.Join(dir, "BUILD")))
}

func TestInitGoModReusesExistingFilegroup(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example_module\n"), 0644))

	existing := `filegroup(
    name = "modfile",
    srcs = ["go.mod"],
    visibility = ["PUBLIC"],
)
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "BUILD"), []byte(existing), 0644))

	label, err := initGoMod(dir)
	require.NoError(t, err)
	assert.Equal(t, "//:modfile", label)

	// The existing build file should be left alone.
	b, err := os.ReadFile(filepath.Join(dir, "BUILD"))
	require.NoError(t, err)
	assert.Equal(t, existing, string(b))
}
