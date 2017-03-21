package gc

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

func TestRewriteFile(t *testing.T) {
	state := core.NewBuildState(0, nil, 4, core.DefaultConfiguration())
	// Copy file to avoid any issues with links etc.
	wd, _ := os.Getwd()
	err := core.CopyFile("src/gc/test_data/before.build", path.Join(wd, "test.build"), 0644)
	assert.NoError(t, err)
	assert.NoError(t, RewriteFile(state, "test.build", []string{"prometheus", "cover"}))
	rewritten, err := ioutil.ReadFile("test.build")
	assert.NoError(t, err)
	after, err := ioutil.ReadFile("src/gc/test_data/after.build")
	assert.NoError(t, err)
	assert.EqualValues(t, string(after), string(rewritten))
}
