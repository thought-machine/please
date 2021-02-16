// package main_test has some basic tests to check please_go_install works but it's quite hard to build up real world
// examples here so the majority of coverage comes from //test/go_modules
package main_test

import (
	"bytes"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thought-machine/please/src/fs"
)


func TestMissingImport(t *testing.T) {
	cmd := compile("missing_import")

	stdOut := &bytes.Buffer{}
	cmd.Stdout = stdOut

	_ = cmd.Run()
	assert.Equal(t, "_build/example.com/missing_import/missing_import.go:3:8: can't find import: \"github.com/doesnt-exist\"\n", stdOut.String())
}

func TestNoSources(t *testing.T) {
	cmd := compile("no_sources")

	stdErr := &bytes.Buffer{}
	cmd.Stderr = stdErr

	err := cmd.Run()
	require.Error(t, err)
	assert.Contains(t, stdErr.String(), "Failed to compile example.com/no_sources: no buildable Go source files in test_data/example.com/no_sources")
}

func TestLocalImports(t *testing.T) {
	cmd := compile("local_imports/foo")

	stdErr := &bytes.Buffer{}
	cmd.Stderr = stdErr

	err := cmd.Run()
	require.NoError(t, err)

	expectedOut := "tools/please_go_install/out/example.com/local_imports/foo.a"
	require.True(t, fs.FileExists(expectedOut), "output file %s wasn't created", expectedOut)
}

func compile(pkgs ...string) *exec.Cmd {
	args := []string{"-r", "test_data/example.com", "-n", "example.com", "-i", "test_data/empty.importcfg", "-g", "go", "-o", "out"}
	cmd := exec.Command("./please_go_install", append(args, pkgs...)...)
	cmd.Dir = "tools/please_go_install"
	return cmd
}

func TestMain(m *testing.M) {
	f, err := os.Create("tools/please_go_install/test_data/empty.importcfg")
	if err != nil {
		panic(err)
	}
	f.Close()

	os.Exit(m.Run())
}