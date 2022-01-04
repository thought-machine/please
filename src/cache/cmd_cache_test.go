package cache

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thought-machine/please/src/core"
)

func init() {
	os.Chdir("src/cache/test_data")
}

func TestCmdStoreAndRetrieve(t *testing.T) {
	target := core.NewBuildTarget(core.NewBuildLabel("pkg/name", "label_name"))
	target.AddOutput("testfile2")
	config := core.DefaultConfiguration()
	config.Cache.StoreCommand = "cat > $CACHE_KEY;  >&2 echo Store $CACHE_KEY"
	config.Cache.RetrieveCommand = ">&2 echo Retrieve $CACHE_KEY; cat $CACHE_KEY"
	cache := newCmdCache(config)

	key := []byte("test_key")
	os.Chdir("src/cache/test_data")
	cache.Store(target, key, target.Outputs())

	b, err := ioutil.ReadFile("plz-out/gen/pkg/name/testfile2")
	assert.NoError(t, err)

	// Remove the file before we retrieve
	metadata := cache.Retrieve(target, key, nil)
	assert.NotNil(t, metadata)

	b2, err := ioutil.ReadFile("plz-out/gen/pkg/name/testfile2")
	assert.NoError(t, err)
	assert.Equal(t, b, b2)
}
