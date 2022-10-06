package cache

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

func init() {
	os.Chdir("src/cache/test_data")
}

func TestCmdRetrieveInvalidCommand(t *testing.T) {
	target := core.NewBuildTarget(core.NewBuildLabel("pkg/name", "label_name"))
	target.AddOutput("testfile2")
	config := core.DefaultConfiguration()
	config.Cache.RetrieveCommand = "/xxbin/xx_no_such_command"
	cache := newCmdCache(config)

	key := []byte("TestCmdRetrieveInvalidCommand")
	os.Chdir("src/cache/test_data")

	hit := cache.Retrieve(target, key, nil)
	// expected to fail because command does not exist
	assert.False(t, hit)
}

func TestCmdRetrieveExitCode(t *testing.T) {
	target := core.NewBuildTarget(core.NewBuildLabel("pkg/name", "label_name"))
	target.AddOutput("testfile2")
	config := core.DefaultConfiguration()
	config.Cache.RetrieveCommand = "echo rubbish; exit 1"
	cache := newCmdCache(config)

	key := []byte("TestCmdRetrieveExitCode")
	os.Chdir("src/cache/test_data")

	hit := cache.Retrieve(target, key, nil)
	// expected to fail because of exit 1
	assert.False(t, hit)
}

func TestCmdRetrieveNoHit(t *testing.T) {
	target := core.NewBuildTarget(core.NewBuildLabel("pkg/name", "label_name"))
	target.AddOutput("testfile2")
	config := core.DefaultConfiguration()
	config.Cache.RetrieveCommand = "cat XXX_no_such_key"
	cache := newCmdCache(config)

	key := []byte("TestCmdRetrieveNoHit")
	os.Chdir("src/cache/test_data")

	hit := cache.Retrieve(target, key, nil)
	// expected to fail because of non-exiting cache entry
	assert.False(t, hit)
}

func TestCmdStoreInvalidCommand(t *testing.T) {
	target := core.NewBuildTarget(core.NewBuildLabel("pkg/name", "label_name"))
	target.AddOutput("testfile2")
	config := core.DefaultConfiguration()
	config.Cache.StoreCommand = "/xxbin/xx_no_such_command"
	config.Cache.RetrieveCommand = "cat $CACHE_KEY"
	cache := newCmdCache(config)

	key := []byte("TestCmdStoreAndRetrieve")
	os.Chdir("src/cache/test_data")

	// cache interface does not provide any result here...
	// but we should at least not panic or that alike
	cache.Store(target, key, target.Outputs())
}

func TestCmdStoreAndRetrieve(t *testing.T) {
	target := core.NewBuildTarget(core.NewBuildLabel("pkg/name", "label_name"))
	target.AddOutput("testfile2")
	config := core.DefaultConfiguration()
	config.Cache.StoreCommand = "cat > $CACHE_KEY;  >&2 echo Store $CACHE_KEY"
	config.Cache.RetrieveCommand = ">&2 echo Retrieve $CACHE_KEY; cat $CACHE_KEY"
	cache := newCmdCache(config)

	key := []byte("TestCmdStoreAndRetrieve")
	os.Chdir("src/cache/test_data")
	cache.Store(target, key, target.Outputs())

	b, err := os.ReadFile("plz-out/gen/pkg/name/testfile2")
	assert.NoError(t, err)

	hit := cache.Retrieve(target, key, nil)
	assert.True(t, hit)

	b2, err := os.ReadFile("plz-out/gen/pkg/name/testfile2")
	assert.NoError(t, err)
	assert.Equal(t, b, b2)
}

func TestCmdStoreAndRetrieveExitCode(t *testing.T) {
	target := core.NewBuildTarget(core.NewBuildLabel("pkg/name", "label_name"))
	target.AddOutput("testfile2")
	config := core.DefaultConfiguration()
	config.Cache.StoreCommand = "cat > $CACHE_KEY"
	config.Cache.RetrieveCommand = "cat $CACHE_KEY && exit 1"
	cache := newCmdCache(config)

	key := []byte("TestCmdStoreAndRetrieveExitCode")
	os.Chdir("src/cache/test_data")
	cache.Store(target, key, target.Outputs())

	hit := cache.Retrieve(target, key, nil)
	// expected to fail because of "exit 1"
	assert.False(t, hit)
}
