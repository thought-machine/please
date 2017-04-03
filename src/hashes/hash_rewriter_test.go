package hashes

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"

	"core"
)

func init() {
	// Move the parser engine .so files into the current directory so we find them.
	const dir = "src/parse/cffi"
	info, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatalf("%s", err)
	}
	for _, inf := range info {
		if err := os.Rename(path.Join(dir, inf.Name()), inf.Name()); err != nil {
			log.Fatalf("%s", err)
		}
	}
}

func TestRewriteHashes(t *testing.T) {
	state := core.NewBuildState(0, nil, 4, core.DefaultConfiguration())
	// Copy file to avoid any issues with links etc.
	wd, _ := os.Getwd()
	err := core.CopyFile("src/hashes/test_data/before.build", path.Join(wd, "test.build"), 0644)
	assert.NoError(t, err)
	assert.NoError(t, rewriteHashes(state, "test.build", "test_x86", map[string]string{
		"test1": "b9643f8154a9e9912d730a931d329afc82a44a52",
		"test2": "bd79dd61c1494072271f3d13350ccbc26c25a09e",
		"test3": "94ead0b0422cad925910e5f8b6f9bd93b309f8f0",
	}))
	rewritten, err := ioutil.ReadFile("test.build")
	assert.NoError(t, err)
	after, err := ioutil.ReadFile("src/hashes/test_data/after.build")
	assert.NoError(t, err)
	assert.EqualValues(t, string(after), string(rewritten))
}
