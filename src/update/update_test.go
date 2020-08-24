package update

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
)

var server *httptest.Server

type fakeLogBackend struct{}

func (*fakeLogBackend) Log(level logging.Level, calldepth int, rec *logging.Record) error {
	if level == logging.CRITICAL {
		panic(rec.Message())
	}
	fmt.Printf("%s\n", rec.Message())
	return nil
}

func TestVerifyNewPlease(t *testing.T) {
	assert.True(t, verifyNewPlease("github.com/thought-machine/please/src/please", core.PleaseVersion.String()))
	assert.False(t, verifyNewPlease("github.com/thought-machine/please/src/please", "wibble"))
	assert.False(t, verifyNewPlease("wibble", core.PleaseVersion.String()))
}

func TestFindLatestVersion(t *testing.T) {
	assert.Equal(t, "42.0.0", findLatestVersion(server.URL).String())
	assert.Panics(t, func() { findLatestVersion(server.URL + "/blah") })
	assert.Panics(t, func() { findLatestVersion("notaurl") })
}

func TestFileMode(t *testing.T) {
	assert.Equal(t, os.FileMode(0775), fileMode("please"))
	assert.Equal(t, os.FileMode(0664), fileMode("junit_runner.jar"))
	assert.Equal(t, os.FileMode(0664), fileMode("libplease_parser_pypy.so"))
}

func TestLinkNewFile(t *testing.T) {
	c := makeConfig("linknewfile")
	dir := path.Join(c.Please.Location, c.Please.Version.String())
	assert.NoError(t, os.MkdirAll(dir, core.DirPermissions))
	assert.NoError(t, ioutil.WriteFile(path.Join(dir, "please"), []byte("test"), 0775))
	linkNewFile(c, "please")
	assert.True(t, core.PathExists(path.Join(c.Please.Location, "please")))
	assert.NoError(t, ioutil.WriteFile(path.Join(c.Please.Location, "exists"), []byte("test"), 0775))
}

func TestDownloadNewPlease(t *testing.T) {
	c := makeConfig("downloadnewplease")
	downloadPlease(c, false, true)
	// Should have written new file
	assert.True(t, core.PathExists(path.Join(c.Please.Location, c.Please.Version.String(), "please")))
	// Should not have written this yet though
	assert.False(t, core.PathExists(path.Join(c.Please.Location, "please")))
	// Panics because it's not a valid .tar.gz
	c.Please.Version.UnmarshalFlag("1.0.0")
	assert.Panics(t, func() { downloadPlease(c, false, true) })
	// Panics because it doesn't exist
	c.Please.Version.UnmarshalFlag("2.0.0")
	assert.Panics(t, func() { downloadPlease(c, false, true) })
	// Panics because invalid URL
	c.Please.DownloadLocation = "notaurl"
	assert.Panics(t, func() { downloadPlease(c, false, true) })
}

func TestShouldUpdateVersionsMatch(t *testing.T) {
	c := makeConfig("shouldupdate")
	c.Please.Version.Set(core.PleaseVersion.String())
	// Versions match, update is never needed
	assert.False(t, shouldUpdate(c, false, false))
	assert.False(t, shouldUpdate(c, true, true))
}

func TestShouldUpdateVersionsDontMatch(t *testing.T) {
	c := makeConfig("shouldupdate")
	c.Please.Version.UnmarshalFlag("2.0.0")
	// Versions don't match but update is skipped
	assert.False(t, shouldUpdate(c, false, false))
	// Versions don't match, update is not skipped.
	assert.True(t, shouldUpdate(c, true, false))
	// Updates are off in config.
	c.Please.SelfUpdate = false
	assert.False(t, shouldUpdate(c, true, false))
}

func TestShouldUpdateGTEVersion(t *testing.T) {
	c := makeConfig("shouldupdate")
	c.Please.Version.UnmarshalFlag(">=2.0.0")
	assert.False(t, shouldUpdate(c, true, false))
	assert.True(t, shouldUpdate(c, true, true))
}

func TestShouldUpdateNoDownloadLocation(t *testing.T) {
	c := makeConfig("shouldupdate")
	// Download location isn't set
	c.Please.DownloadLocation = ""
	assert.False(t, shouldUpdate(c, true, true))
}

func TestShouldUpdateNoPleaseLocation(t *testing.T) {
	c := makeConfig("shouldupdate")
	// Please location isn't set
	c.Please.Location = ""
	assert.False(t, shouldUpdate(c, true, true))
}

func TestShouldUpdateNoVersion(t *testing.T) {
	c := makeConfig("shouldupdate")
	// No version is set, shouldn't update unless we force
	c.Please.Version = cli.Version{}
	assert.False(t, shouldUpdate(c, true, false))
	assert.Equal(t, core.PleaseVersion, c.Please.Version.Semver())
	c.Please.Version = cli.Version{}
	assert.True(t, shouldUpdate(c, true, true))
}

func TestDownloadAndLinkPlease(t *testing.T) {
	c := makeConfig("downloadandlink")
	c.Please.Version.UnmarshalFlag(core.PleaseVersion.String())
	newPlease := downloadAndLinkPlease(c, false, true)
	assert.True(t, core.PathExists(newPlease))
}

func TestDownloadAndLinkPleaseBadVersion(t *testing.T) {
	c := makeConfig("downloadandlink")
	assert.Panics(t, func() { downloadAndLinkPlease(c, false, true) })
	// Should have deleted the thing it downloaded.
	assert.False(t, core.PathExists(path.Join(c.Please.Location, c.Please.Version.String())))
}

func TestFilterArgs(t *testing.T) {
	assert.Equal(t, []string{"plz", "update"}, filterArgs(false, []string{"plz", "update"}))
	assert.Equal(t, []string{"plz", "update", "--force"}, filterArgs(false, []string{"plz", "update", "--force"}))
	assert.Equal(t, []string{"plz", "update"}, filterArgs(true, []string{"plz", "update", "--force"}))
}

func TestXZVersions(t *testing.T) {
	var v cli.Version
	v.UnmarshalFlag("13.1.9")
	assert.False(t, shouldUseXZ(v))
	v.UnmarshalFlag("13.2.0")
	assert.True(t, shouldUseXZ(v))
}

func handler(w http.ResponseWriter, r *http.Request) {
	vCurrent := fmt.Sprintf("/%s_%s/%s/please_%s.tar.xz", runtime.GOOS, runtime.GOARCH, core.PleaseVersion, core.PleaseVersion)
	v42 := fmt.Sprintf("/%s_%s/42.0.0/please_42.0.0.tar.xz", runtime.GOOS, runtime.GOARCH)
	if r.URL.Path == "/latest_version" {
		w.Write([]byte("42.0.0"))
	} else if r.URL.Path == vCurrent || r.URL.Path == v42 {
		b, err := ioutil.ReadFile("src/update/please_test.tar.xz")
		if err != nil {
			panic(err)
		}
		w.Write(b)
	} else if r.URL.Path == fmt.Sprintf("/%s_%s/1.0.0/please_1.0.0.tar.gz", runtime.GOOS, runtime.GOARCH) {
		w.Write([]byte("notatarball"))
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func makeConfig(dir string) *core.Configuration {
	c := core.DefaultConfiguration()
	wd, _ := os.Getwd()
	c.Please.Location = path.Join(wd, dir)
	c.Please.DownloadLocation.UnmarshalFlag(server.URL)
	c.Please.Version.UnmarshalFlag("42.0.0")
	return c
}

func TestMain(m *testing.M) {
	// Reset this so it panics instead of exiting on Fatal messages
	logging.SetBackend(&fakeLogBackend{})
	server = httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()
	os.Exit(m.Run())
}
