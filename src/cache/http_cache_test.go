package cache

import (
	"io/ioutil"
	"net/http/httptest"
	"path"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"cache/server"
	"core"
)

var (
	label     core.BuildLabel
	target    *core.BuildTarget
	httpcache *httpCache
	key       []byte
	osName    string
)

func init() {
	osName = runtime.GOOS + "_" + runtime.GOARCH
	label = core.NewBuildLabel("pkg/name", "label_name")
	target = core.NewBuildTarget(label)

	// Arbitrary large numbers so the cleaner never needs to run.
	cache := server.NewCache("src/cache/test_data", 20*time.Hour, 100000, 100000000, 1000000000)
	key, _ = ioutil.ReadFile("src/cache/test_data/testfile")
	testServer := httptest.NewServer(server.BuildRouter(cache))

	config := core.DefaultConfiguration()
	config.Cache.HttpUrl.UnmarshalFlag(testServer.URL)
	config.Cache.HttpWriteable = true
	httpcache = newHttpCache(config)
}

func TestStore(t *testing.T) {
	target.AddOutput("testfile")
	httpcache.Store(target, []byte("test_key"))
	abs, _ := filepath.Abs(path.Join("src/cache/test_data", osName, "pkg/name", "label_name"))
	if !core.PathExists(abs) {
		t.Errorf("Test file %s was not stored in cache.", abs)
	}
}

func TestRetrieve(t *testing.T) {
	if !httpcache.Retrieve(target, []byte("test_key")) {
		t.Error("Artifact expected and not found.")
	}
}

func TestClean(t *testing.T) {
	httpcache.Clean(target)
	filename := path.Join("src/cache/test_data", osName, "pkg/name/label_name")
	if core.PathExists(filename) {
		t.Errorf("File %s was not removed from cache.", filename)
	}
}
