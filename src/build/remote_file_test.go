package build

import (
	"net"
	"net/http"
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

// writeHomeSecret puts the secret the tests read at ~/secret, with the home directory pointed
// somewhere this test owns. Writing to the real one would leave a file behind, and it is left
// read-only, which on Windows means the next run cannot replace it.
func writeHomeSecret(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // What os.UserHomeDir reads on Windows.
	require.NoError(t, fs.CopyFile("secret", fs.ExpandHomePath("~/secret"), 0444))
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

	writeHomeSecret(t)

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

	require.NoError(t, fetchRemoteFile(state, target))
}

func TestBasicAuth(t *testing.T) {
	state, target := newState("//pkg:header_test")
	target.IsRemoteFile = true
	target.Sources = []core.BuildInput{core.URLLabel("http://localhost:8080/header")}
	target.AddOutput("header")
	target.AddLabel("remote_file:username:foo")
	target.AddLabel("remote_file:password_file:~/secret")

	writeHomeSecret(t)

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

	require.NoError(t, fetchRemoteFile(state, target))
}
