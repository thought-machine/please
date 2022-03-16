// package main_test has some basic tests to check please_go_install works but it's quite hard to build up real world
// examples here so the majority of coverage comes from //test/go_modules
package install

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/tools/please_go/install/exec"
)

func TestMissingImport(t *testing.T) {
	install, stdOut, _ := newInstall()
	err := install.Install([]string{"missing_import"})
	require.Error(t, err)
	assert.Contains(t, []string{
		filepath.Join(os.Getenv("TMP_DIR"), "tools/please_go/install/test_data/example.com/missing_import/missing_import.go:3:8: could not import github.com/doesnt-exist (open : no such file or directory)\n"),    // go 1.18
		filepath.Join(os.Getenv("TMP_DIR"), "tools/please_go/install/test_data/example.com/missing_import/missing_import.go:3:16: could not import github.com/doesnt-exist (open : no such file or directory)\n"),   // go 1.18 (sometimes the column is different?)
		filepath.Join(os.Getenv("TMP_DIR"), "tools/please_go/install/test_data/example.com/missing_import/missing_import.go:3:8: could not import \"github.com/doesnt-exist\": open : no such file or directory\n"), // go 1.17
		filepath.Join(os.Getenv("TMP_DIR"), "tools/please_go/install/test_data/example.com/missing_import/missing_import.go:3:8: can't find import: \"github.com/doesnt-exist\"\n"),                                 // go 1.16
	}, stdOut.String())
}

func TestNoSources(t *testing.T) {
	install, _, _ := newInstall()

	err := install.Install([]string{"no_sources"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to compile example.com/no_sources: no buildable Go source files")
}

func TestLocalImports(t *testing.T) {
	install, _, _ := newInstall()

	err := install.Install([]string{"local_imports/foo"})
	require.NoError(t, err)

	expectedOut := "out/example.com/local_imports/foo/foo.a"
	require.True(t, fs.FileExists(expectedOut), "output file %s wasn't created", expectedOut)
}

func newInstall() (*PleaseGoInstall, *bytes.Buffer, *bytes.Buffer) {
	install := New([]string{}, "tools/please_go/install/test_data/example.com", "example.com", "tools/please_go/install/test_data/empty.importcfg", "", "", "go", "cc", "pkg-config", "out", "")

	stdOut := &bytes.Buffer{}
	stdIn := &bytes.Buffer{}
	install.tc.Exec = &exec.Executor{
		Stdout: stdOut,
		Stderr: stdIn,
	}
	return install, stdOut, stdIn
}

func TestMain(m *testing.M) {
	f, err := os.Create("tools/please_go/install/test_data/empty.importcfg")
	if err != nil {
		panic(err)
	}
	f.Close()

	os.Exit(m.Run())
}
