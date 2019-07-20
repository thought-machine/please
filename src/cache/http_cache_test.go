package cache

import (
	"io/ioutil"
	"net/http/httptest"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/tools/cache/server"
)

var (
	label     core.BuildLabel
	target    *core.BuildTarget
	httpcache *httpCache
	key       []byte
)

func init() {
	label = core.NewBuildLabel("pkg/name", "label_name")
	target = core.NewBuildTarget(label)

	// Arbitrary large numbers so the cleaner never needs to run.
	cache := server.NewCache("src/cache/test_data", 20*time.Hour, 100000, 100000000, 1000000000)
	key, _ = ioutil.ReadFile("src/cache/test_data/testfile")
	testServer := httptest.NewServer(server.BuildRouter(cache))

	config := core.DefaultConfiguration()
	config.Cache.HTTPURL.UnmarshalFlag(testServer.URL)
	config.Cache.HTTPWriteable = true
	httpcache = newHTTPCache(config)
}

func TestStore(t *testing.T) {
	target.AddOutput("testfile")
	httpcache.Store(target, []byte("test_key"), &core.BuildMetadata{}, target.Outputs())
	abs, _ := filepath.Abs(path.Join("src/cache/test_data", core.OsArch, "pkg/name", "label_name"))
	if !core.PathExists(abs) {
		t.Errorf("Test file %s was not stored in cache.", abs)
	}
}

func TestRetrieve(t *testing.T) {
	if httpcache.Retrieve(target, []byte("test_key")) == nil {
		t.Error("Artifact expected and not found.")
	}
}

func TestClean(t *testing.T) {
	httpcache.Clean(target)
	filename := path.Join("src/cache/test_data", core.OsArch, "pkg/name/label_name")
	if core.PathExists(filename) {
		t.Errorf("File %s was not removed from cache.", filename)
	}
}
