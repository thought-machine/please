package cache

import (
	"os"
	"path"
	"runtime"
	"testing"
	"time"

	"cache/server"
	"core"
)

var (
	label    core.BuildLabel
	rpccache *rpcCache
	osName   string
)

func init() {
	runtime.GOMAXPROCS(1) // Don't allow tests to run in parallel, they should work but makes debugging tricky
	osName = runtime.GOOS + "_" + runtime.GOARCH
	label = core.NewBuildLabel("pkg/name", "label_name")

	// Move this directory from test_data to somewhere local.
	// This is easier than changing our working dir & a better test of some things too.
	if err := os.Rename("src/cache/test_data/plz-out", "plz-out"); err != nil {
		log.Fatalf("Failed to prepare test directory: %s\n", err)
	}
	// Arbitrary large numbers so the cleaner never needs to run.
	cache := server.NewCache("src/cache/test_data", 100000, 100000000, 1000000000)

	go server.ServeGrpcForever(7677, cache)

	config := core.DefaultConfiguration()
	config.Cache.RpcUrl = "localhost:7677"
	config.Cache.RpcWriteable = true

	var err error
	rpccache, err = newRpcCache(config)
	if err != nil {
		log.Fatalf("Failed to create RPC cache: %s", err)
	}

	// Busy-wait sucks but this isn't supposed to be visible from outside.
	for i := 0; i < 100 && !rpccache.Connected; i++ {
		time.Sleep(100 * time.Millisecond)
	}
}

func TestStore(t *testing.T) {
	target := core.NewBuildTarget(label)
	target.AddOutput("testfile2")
	rpccache.Store(target, []byte("test_key"))
	expectedPath := path.Join("src/cache/test_data", osName, "pkg/name", "label_name", "dGVzdF9rZXk", target.Outputs()[0])
	if !core.PathExists(expectedPath) {
		t.Errorf("Test file %s was not stored in cache.", expectedPath)
	}
}

func TestRetrieve(t *testing.T) {
	target := core.NewBuildTarget(label)
	target.AddOutput("testfile")
	if !rpccache.Retrieve(target, []byte("test_key")) {
		t.Error("Artifact expected and not found.")
	}
}

func TestStoreAndRetrieve(t *testing.T) {
	target := core.NewBuildTarget(label)
	target.AddOutput("testfile3")
	rpccache.Store(target, []byte("test_key"))
	// Remove the file so we can test retrieval correctly
	outPath := path.Join(target.OutDir(), target.Outputs()[0])
	if err := os.Remove(outPath); err != nil {
		t.Errorf("Failed to remove artifact: %s", err)
	}
	if !rpccache.Retrieve(target, []byte("test_key")) {
		t.Error("Artifact expected and not found.")
	} else if !core.PathExists(outPath) {
		t.Errorf("Artifact %s doesn't exist after alleged cache retrieval", outPath)
	}
}

func TestClean(t *testing.T) {
	target := core.NewBuildTarget(label)
	rpccache.Clean(target)
	filename := path.Join("src/cache/test_data", osName, "pkg/name/label_name")
	if core.PathExists(filename) {
		t.Errorf("File %s was not removed from cache.", filename)
	}
}
