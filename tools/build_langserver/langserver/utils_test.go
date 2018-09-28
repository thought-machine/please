package langserver

import (
	"testing"
	"path"
	"github.com/stretchr/testify/assert"
	"tools/build_langserver/lsp"
	"core"
)

func TestIsURL(t *testing.T) {
	currentFile, err := getFileinCwd("utils_test.go")
	assert.Equal(t, err, nil)

	assert.False(t, IsURL(currentFile))

	documentUri := "file://" + currentFile
	assert.True(t, IsURL(documentUri))
}

func TestGetPathFromURL(t *testing.T) {
	currentFile, err := getFileinCwd("utils_test.go")
	assert.Equal(t, err, nil)
	documentUri := lsp.DocumentURI("file://" + currentFile)

	// Test GetPathFromURL when documentURI passed in is a URI
	p, err :=  GetPathFromURL(documentUri, "file")
	assert.Equal(t, err, nil)
	assert.Equal(t, p, string(currentFile))

	// Test GetPathFromURL when documentURI passed in is a file path
	p2, err := GetPathFromURL(currentFile, "File")
	assert.Equal(t, err, nil)
	assert.Equal(t, p2, string(currentFile))

}

func TestGetPathFromURLFail(t *testing.T) {
	// Test invalid file fail with Bogus file
	bogusFile, err := getFileinCwd("BLAH.blah")
	assert.Equal(t, err, nil)

	p, err := GetPathFromURL(bogusFile, "file")
	assert.Equal(t, p, "")
	assert.Error(t, err)

	// Test invalid file not in repo root
	p2, err := GetPathFromURL(lsp.DocumentURI("/tmp"), "path")
	assert.Equal(t, p2, "")
	assert.Error(t, err)

	// Test invalid pathtype
	currentFile, err := getFileinCwd("utils_test.go")
	assert.Equal(t, err, nil)
	p3, err := GetPathFromURL(currentFile, "dir")
	assert.Equal(t, p3, "")
	assert.Error(t, err)
}


func TestEnsureURL(t *testing.T) {
	currentFile, err := getFileinCwd("utils_test.go")
	assert.Equal(t, err, nil)

	uri, err := EnsureURL(currentFile, "file")
	assert.Equal(t, err, nil)
	assert.Equal(t, uri, lsp.DocumentURI("file://" + string(currentFile)))
}

/*
 * Utilities for tests in this file
 */
func getFileinCwd(name string) (lsp.DocumentURI, error) {
	core.FindRepoRoot()
	filePath := path.Join(core.RepoRoot, "tools/build_langserver/langserver/" + name)

	return lsp.DocumentURI(filePath), nil
}