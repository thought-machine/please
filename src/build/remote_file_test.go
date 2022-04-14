package build

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"net/http"
	"testing"
)

func Server() (*http.Server, *http.ServeMux) {
	s := &http.Server{Addr: ":8080", Handler: http.NewServeMux()}
	return s, s.Handler.(*http.ServeMux)
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

	go s.ListenAndServe()

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

	go s.ListenAndServe()

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

	go s.ListenAndServe()

	err = fetchRemoteFile(state, target)
	require.NoError(t, err)
}
