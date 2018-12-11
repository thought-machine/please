package server

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

var (
	server       *httptest.Server
	realURL      string
	fakeURL      string
	otherRealURL string
	extraRealURL string
	reader       io.Reader
)

const cachePath = "tools/cache/server/test_data"

func init() {
	c := newCache(cachePath)
	server = httptest.NewServer(BuildRouter(c))
	realURL = fmt.Sprintf("%s/artifact/darwin_amd64/pack/label/hash/label.ext", server.URL)
	otherRealURL = fmt.Sprintf("%s/artifact/linux_amd64/otherpack/label/hash/label.ext", server.URL)
	extraRealURL = fmt.Sprintf("%s/artifact/extrapack/label", server.URL)
	fakeURL = fmt.Sprintf("%s/artifact/darwin_amd64/somepack/somelabel/somehash/somelabel.ext", server.URL)
}

func TestOutputFolderExists(t *testing.T) {
	if !core.PathExists(cachePath) {
		t.Errorf("%s does not exist.", cachePath)
	}
}

func TestGetHandler(t *testing.T) {
	request, err := http.NewRequest("GET", realURL, reader)
	assert.NoError(t, err)
	res, err := http.DefaultClient.Do(request)

	if err != nil {
		t.Error(err)
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		t.Error("Expected response Status Accepted, got:", res.Status)
	}
}

func TestGetHandlerError(t *testing.T) {
	request, _ := http.NewRequest("GET", fakeURL, reader)
	res, _ := http.DefaultClient.Do(request)
	if res.StatusCode != 404 {
		t.Error("Expected nil and found artifact.")
	}
}

func TestPostHandler(t *testing.T) {
	fileContent := "This is a newly created file."
	reader = strings.NewReader(fileContent)
	request, err := http.NewRequest("POST", fakeURL, reader)
	assert.NoError(t, err)
	res, err := http.DefaultClient.Do(request)

	if err != nil {
		t.Error(err)
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		t.Error("Expected response Status Accepted, got:", res.Status)
	}
}

func TestDeleteHandler(t *testing.T) {
	request, _ := http.NewRequest("DELETE", extraRealURL, reader)
	res, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Error(err)
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		t.Error("Expected response Status Accepted, got:", res.Status)
	}
}

func TestDeleteAllHandler(t *testing.T) {
	request, _ := http.NewRequest("DELETE", server.URL, reader)
	res, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Error(err)
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		t.Error("Expected response Status Accepted, got:", res.Status)
	}
}
