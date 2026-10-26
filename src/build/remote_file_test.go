package build

import (
	"encoding/pem"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

func Server() (*http.Server, *http.ServeMux) {
	s := &http.Server{Addr: ":8080", Handler: http.NewServeMux()}
	return s, s.Handler.(*http.ServeMux)
}

func listen(s *http.Server) net.Listener {
	lis, err := net.Listen("tcp", s.Addr)
	if err != nil {
		log.Fatalf("Failed to listen: %s", err)
	}
	return lis
}

func TestHeader(t *testing.T) {
	state, target := newState("//pkg:header_test")
	target.IsRemoteFile = true
	target.Sources = []core.BuildInput{core.URLLabel("http://localhost:8080/header")}
	target.AddOutput("header")
	target.AddLabel("remote_file:header:foo:fooval")

	s, m := Server()
	m.HandleFunc("/header", func(writer http.ResponseWriter, request *http.Request) {
		foo := request.Header.Get("foo")
		assert.Equal(t, foo, "fooval")
	})
	defer s.Close()
	lis := listen(s)
	go s.Serve(lis)

	err := fetchRemoteFile(state, target)
	require.NoError(t, err)
}

func TestSecretHeader(t *testing.T) {
	state, target := newState("//pkg:header_test")
	target.IsRemoteFile = true
	target.Sources = []core.BuildInput{core.URLLabel("http://localhost:8080/header")}
	target.AddOutput("header")
	target.AddLabel("remote_file:secret_header:foo:~/secret")
	target.AddLabel("remote_file:secret_header:bar:secret")

	err := fs.CopyFile("secret", fs.ExpandHomePath("~/secret"), 0444)
	require.NoError(t, err)

	s, m := Server()
	m.HandleFunc("/header", func(writer http.ResponseWriter, request *http.Request) {
		foo := request.Header.Get("foo")
		assert.Equal(t, foo, "secret val")

		bar := request.Header.Get("bar")
		assert.Equal(t, bar, "secret val")
	})
	defer s.Close()
	lis := listen(s)
	go s.Serve(lis)

	err = fetchRemoteFile(state, target)
	require.NoError(t, err)
}

func TestBasicAuth(t *testing.T) {
	state, target := newState("//pkg:header_test")
	target.IsRemoteFile = true
	target.Sources = []core.BuildInput{core.URLLabel("http://localhost:8080/header")}
	target.AddOutput("header")
	target.AddLabel("remote_file:username:foo")
	target.AddLabel("remote_file:password_file:~/secret")

	err := fs.CopyFile("secret", fs.ExpandHomePath("~/secret"), 0444)
	require.NoError(t, err)

	s, m := Server()
	m.HandleFunc("/header", func(writer http.ResponseWriter, request *http.Request) {
		usr, pass, ok := request.BasicAuth()
		require.True(t, ok)
		assert.Equal(t, "foo", usr)
		assert.Equal(t, "secret val", pass)
	})
	defer s.Close()
	lis := listen(s)
	go s.Serve(lis)

	err = fetchRemoteFile(state, target)
	require.NoError(t, err)
}

func TestAdditionalCACerts(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte("wibble"))
	}))
	defer s.Close()
	// t.TempDir() is relative when run remotely (TMPDIR is .), and relative paths are resolved against RepoRoot.
	certFile, err := filepath.Abs(filepath.Join(t.TempDir(), "ca.pem"))
	require.NoError(t, err)
	err = os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: s.Certificate().Raw}), 0644)
	require.NoError(t, err)

	// The client is only configured once, so reset it to pick up config changes.
	resetHTTPClient := func() {
		httpClientOnce = sync.Once{}
		httpClientErr = nil
	}
	resetHTTPClient()
	t.Cleanup(resetHTTPClient)

	state, target := newState("//pkg:additional_ca_certs_test")
	target.IsRemoteFile = true
	target.Sources = []core.BuildInput{core.URLLabel(s.URL + "/file")}
	target.AddOutput("file")

	err = fetchRemoteFile(state, target)
	assert.ErrorContains(t, err, "certificate")

	resetHTTPClient()
	state.Config.Build.AdditionalCACerts = []string{certFile}
	err = fetchRemoteFile(state, target)
	require.NoError(t, err)
	contents, err := os.ReadFile(filepath.Join(target.TmpDir(), "file"))
	require.NoError(t, err)
	assert.Equal(t, "wibble", string(contents))
}
