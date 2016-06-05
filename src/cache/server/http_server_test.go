package server

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"core"
)

var (
	server       *httptest.Server
	realURL      string
	fakeURL      string
	otherRealURL string
	extraRealURL string
	reader       io.Reader
)

const cachePath = "plz-cache"

func init() {
	c := newCache(cachePath)
	server = httptest.NewServer(BuildRouter(c))
	if !core.PathExists(cachePath + "/darwin_amd64/pack/label/hash/") {
		_ = os.MkdirAll(cachePath+"/darwin_amd64/pack/label/hash/", core.DirPermissions)
		_, _ = os.Create(cachePath + "/darwin_amd64/pack/label/hash/" + "label.ext")
	}
	if !core.PathExists(cachePath + "/linux_amd64/otherpack/label/hash/") {
		_ = os.MkdirAll(cachePath+"/linux_amd64/otherpack/label/hash/", core.DirPermissions)
		_, _ = os.Create(cachePath + "/linux_amd64/otherpack/label/hash/" + "label.ext")
	}
	if !core.PathExists(cachePath + "/linux_amd64/extrapack/label/hash/") {
		_ = os.MkdirAll(cachePath+"/linux_amd64/extrapack/label/hash/", core.DirPermissions)
		_, _ = os.Create(cachePath + "/linux_amd64/extrapack/label/hash/" + "label.ext")
	}

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

func TestRetrieve(t *testing.T) {
	artifact, err := RetrieveArtifact("darwin_amd64/pack/label/hash/label.ext")
	if err != nil {
		t.Error(err)
		return
	}
	if artifact == nil {
		t.Error("Expected artifact and found nil.")
	}
}

func TestRetrieveError(t *testing.T) {
	artifact, err := RetrieveArtifact(cachePath + "/darwin_amd64/somepack/somelabel/somehash/somelabel.ext")
	if artifact != nil {
		t.Error("Expected nil and found artifact.")
	}
	if err == nil {
		t.Error("Expected error and found nil.")
	}
}

func TestGetHandler(t *testing.T) {
	request, err := http.NewRequest("GET", realURL, reader)
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

func TestStore(t *testing.T) {
	fileContent := "This is a newly created file."
	reader = strings.NewReader(fileContent)
	key, err := ioutil.ReadAll(reader)

	err = StoreArtifact("/darwin_amd64/somepack/somelabel/somehash/somelabel.ext", key)
	if err != nil {
		t.Error(err)
	}
}

func TestPostHandler(t *testing.T) {
	fileContent := "This is a newly created file."
	reader = strings.NewReader(fileContent)
	request, err := http.NewRequest("POST", fakeURL, reader)
	res, err := http.DefaultClient.Do(request)

	if err != nil {
		t.Error(err)
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		t.Error("Expected response Status Accepted, got:", res.Status)
	}
}

func TestDeleteArtifact(t *testing.T) {
	err := DeleteArtifact("/linux_amd64/otherpack/label")
	if err != nil {
		t.Error(err)
	}
	absPath, _ := filepath.Abs(cachePath + "/linux_amd64/otherpack/label")
	if core.PathExists(absPath) {
		t.Errorf("%s was not removed from cache.", absPath)
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

func TestDeleteAll(t *testing.T) {
	err := DeleteAllArtifacts()
	if err != nil {
		t.Error(err)
	}
	absPath, _ := filepath.Abs(cachePath)
	if files, _ := ioutil.ReadDir(absPath); len(files) != 0 {

		t.Error(files[0].Name())
		t.Error("The cache was not cleaned.")
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
