package hashes

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

func TestRewriteHashes(t *testing.T) {
	state := core.NewDefaultBuildState()
	// Copy file to avoid any issues with links etc.
	wd, _ := os.Getwd()
	err := fs.CopyFile("src/hashes/test_data/before.build", filepath.Join(wd, "test.build"), 0644)
	assert.NoError(t, err)
	assert.NoError(t, rewriteHashes(state, "test.build", "test_x86", map[string]string{
		"test1": "b9643f8154a9e9912d730a931d329afc82a44a52",
		"test2": "bd79dd61c1494072271f3d13350ccbc26c25a09e",
		"test3": "94ead0b0422cad925910e5f8b6f9bd93b309f8f0",
		"test4": "ab2649b7e58f7e32b0c75be95d11e2979399d392",
	}))
	rewritten, err := os.ReadFile("test.build")
	assert.NoError(t, err)
	after, err := os.ReadFile("src/hashes/test_data/after.build")
	assert.NoError(t, err)
	assert.EqualValues(t, string(after), string(rewritten))
}
