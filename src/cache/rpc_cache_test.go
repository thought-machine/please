package cache

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

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

	startServer(7677, "", "", "")
	rpccache = buildClient(7677, "")
}

func startServer(port int, keyFile, certFile, caCertFile string) *grpc.Server {
	// Arbitrary large numbers so the cleaner never needs to run.
	cache := server.NewCache("src/cache/test_data", 20*time.Hour, 100000, 100000000, 1000000000)
	s, lis := server.BuildGrpcServer(port, cache, nil, keyFile, certFile, caCertFile, "", "")
	go s.Serve(lis)
	return s
}

func buildClient(port int, ca string) *rpcCache {
	config := core.DefaultConfiguration()
	config.Cache.RpcUrl.UnmarshalFlag(fmt.Sprintf("localhost:%d", port))
	config.Cache.RpcWriteable = true
	config.Cache.RpcCACert = ca

	cache, err := newRpcCache(config)
	if err != nil {
		log.Fatalf("Failed to create RPC cache: %s", err)
	}

	// Busy-wait sucks but this isn't supposed to be visible from outside.
	for i := 0; i < 10 && !cache.Connected && cache.Connecting; i++ {
		time.Sleep(100 * time.Millisecond)
	}
	return cache
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

func TestDisconnectAfterEnoughErrors(t *testing.T) {
	// Need a separate cache for this so we don't interfere with the other tests.
	s := startServer(7676, "", "", "")
	c := buildClient(7676, "")

	target := core.NewBuildTarget(label)
	target.AddOutput("testfile4")
	key := []byte("test_key")
	c.Store(target, []byte("test_key"))
	assert.True(t, c.Retrieve(target, key))
	s.Stop()
	// Now after we hit the max number of errors it should disconnect.
	for i := 0; i < maxErrors; i++ {
		assert.True(t, c.Connected)
		assert.False(t, c.Retrieve(target, key))
	}
	assert.False(t, c.Connected)
}

func TestLoadCertificates(t *testing.T) {
	_, err := loadAuth("", "src/cache/test_data/cert.pem", "src/cache/test_data/key.pem")
	assert.NoError(t, err, "Trivial case with PEM files already")
	_, err = loadAuth("", "id_rsa.pub", "id_rsa")
	assert.Error(t, err, "Fails because files don't exist")
}

func TestRetrieveSSL(t *testing.T) {
	// Need a separate cache for this so we don't interfere with the other tests.
	s := startServer(7675, "src/cache/test_data/key.pem", "src/cache/test_data/cert_signed.pem", "src/cache/test_data/ca.pem")
	defer s.Stop()
	c := buildClient(7675, "")
	assert.False(t, c.Connected, "Should fail to connect without giving the client a CA cert")
	c = buildClient(7675, "src/cache/test_data/ca.pem")
	assert.True(t, c.Connected, "Connects OK this time")
}
