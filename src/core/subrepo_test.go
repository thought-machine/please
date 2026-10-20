package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thought-machine/please/src/cli"
)

func TestDir(t *testing.T) {
	s := &Subrepo{Name: "repo", Root: "plz-out/gen/repo"}
	assert.Equal(t, "plz-out/gen/repo/package", s.Dir("package"))
}

func TestInitialiseReturnsConfigErrorEveryTime(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".plzconfig"), []byte(":not valid config\n"), 0644))

	state := NewDefaultBuildState().ForSubrepo("subrepo", false)
	subrepo := NewSubrepo(state, "subrepo", dir, nil, cli.HostArch(), false)

	// Several packages in the subrepo can ask for it to be initialised; all of them should see the error,
	// not just the first one.
	assert.Error(t, state.Initialise(subrepo))
	assert.Error(t, state.Initialise(subrepo))
}
