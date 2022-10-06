package cache

import (
	"io"
	"net"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

func init() {
	os.Chdir("src/cache/test_data")
	// Split up the listen and serve parts to avoid race conditions.
	lis, err := net.Listen("tcp", "127.0.0.1:8989")
	if err != nil {
		log.Fatalf("%s", err)
	}
	go func() {
		http.Serve(lis, &testServer{
			data: map[string][]byte{},
		})
	}()
}

func TestStoreAndRetrieveHTTP(t *testing.T) {
	target := core.NewBuildTarget(core.NewBuildLabel("pkg/name", "label_name"))
	target.AddOutput("testfile2")
	config := core.DefaultConfiguration()
	config.Cache.HTTPURL = "http://127.0.0.1:8989"
	config.Cache.HTTPWriteable = true
	cache := newHTTPCache(config)

	key := []byte("test_key")
	cache.Store(target, key, target.Outputs())

	b, err := os.ReadFile("plz-out/gen/pkg/name/testfile2")
	assert.NoError(t, err)

	// Remove the file before we retrieve
	metadata := cache.Retrieve(target, key, nil)
	assert.NotNil(t, metadata)

	b2, err := os.ReadFile("plz-out/gen/pkg/name/testfile2")
	assert.NoError(t, err)
	assert.Equal(t, b, b2)
}

type testServer struct {
	data map[string][]byte
}

func (s *testServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPut {
		b, _ := io.ReadAll(r.Body)
		s.data[r.URL.Path] = b
		w.WriteHeader(http.StatusNoContent)
		return
	}
	data, present := s.data[r.URL.Path]
	if !present {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Write(data)
}
