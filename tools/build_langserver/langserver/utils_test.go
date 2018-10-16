package langserver

import (
	"context"
	"core"
	"fmt"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"tools/build_langserver/lsp"
)

func TestIsURL(t *testing.T) {
	currentFile, err := getFileinCwd("utils_test.go")
	assert.Equal(t, err, nil)

	assert.False(t, IsURL(currentFile))

	documentURI := "file://" + currentFile
	assert.True(t, IsURL(documentURI))
}

func TestGetPathFromURL(t *testing.T) {
	currentFile, err := getFileinCwd("utils_test.go")
	assert.Equal(t, err, nil)
	documentURI := lsp.DocumentURI("file://" + currentFile)

	// Test GetPathFromURL when documentURI passed in is a URI
	p, err := GetPathFromURL(documentURI, "file")
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
	assert.Equal(t, uri, lsp.DocumentURI("file://"+string(currentFile)))
}

func TestReadFile(t *testing.T) {
	ctx := context.Background()
	filepath := path.Join(core.RepoRoot, "tools/build_langserver/langserver/test_data/example.build")
	uri := lsp.DocumentURI("file://" + filepath)

	// Test ReadFile() with filepath as uri
	content, err := ReadFile(ctx, lsp.DocumentURI(filepath))
	// Type checking
	assert.Equal(t, fmt.Sprintf("%T", content), "[]string")
	assert.Equal(t, err, nil)
	// Content check
	assert.Equal(t, content[0], "go_library(")
	assert.Equal(t, strings.TrimSpace(content[1]), "name = \"langserver\",")

	// Test ReadFile() with uri as uri
	content1, err := ReadFile(ctx, uri)
	// Type checking
	assert.Equal(t, fmt.Sprintf("%T", content), "[]string")
	assert.Equal(t, err, nil)
	// Content check
	assert.Equal(t, content1[0], "go_library(")
	assert.Equal(t, strings.TrimSpace(content1[1]), "name = \"langserver\",")

}

func TestReadFileCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	filepath := path.Join(core.RepoRoot, "tools/build_langserver/langserver/test_data/example.build")

	cancel()
	content, err := ReadFile(ctx, lsp.DocumentURI(filepath))
	assert.Equal(t, content, []string(nil))
	assert.Equal(t, err, nil)
}

func TestGetLineContent(t *testing.T) {
	ctx := context.Background()
	filepath := path.Join(core.RepoRoot, "tools/build_langserver/langserver/test_data/example.build")
	pos := lsp.Position{
		Line: 0,
		Character: 0,
	}

	line, err := GetLineContent(ctx, lsp.DocumentURI(filepath), pos)
	assert.Equal(t, err, nil)
	assert.Equal(t, strings.TrimSpace(line[0]), "go_library(")
}

/*
 * Utilities for tests in this file
 */
func getFileinCwd(name string) (lsp.DocumentURI, error) {
	core.FindRepoRoot()
	filePath := path.Join(core.RepoRoot, "tools/build_langserver/langserver/"+name)

	return lsp.DocumentURI(filePath), nil
}
