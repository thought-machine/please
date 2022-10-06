package gc

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

func TestRewriteFile(t *testing.T) {
	state := core.NewDefaultBuildState()
	// Copy file to avoid any issues with links etc.
	wd, _ := os.Getwd()
	err := fs.CopyFile("src/gc/test_data/before.build", path.Join(wd, "test.build"), 0644)
	assert.NoError(t, err)
	assert.NoError(t, RewriteFile(state, "test.build", []string{"prometheus", "cover"}))
	rewritten, err := os.ReadFile("test.build")
	assert.NoError(t, err)
	after, err := os.ReadFile("src/gc/test_data/after.build")
	assert.NoError(t, err)
	assert.EqualValues(t, string(after), string(rewritten))
}
