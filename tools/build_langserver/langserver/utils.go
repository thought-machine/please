package langserver

import (
	"core"
	"errors"
	"fmt"
	"fs"
	"path/filepath"
	"strings"
	"tools/build_langserver/lsp"
)

// IsURL checks if the documentUri passed has 'file://' prefix
func IsURL(uri lsp.DocumentURI) bool {
	return strings.HasPrefix(string(uri), "file://")
}

// EnsureURL ensures that the documentURI is a valid path in the filesystem and a valid 'file://' URI
func EnsureURL(uri lsp.DocumentURI, pathType string) (url lsp.DocumentURI, err error) {
	documentPath, err := GetPathFromURL(uri, pathType)
	if err != nil {
		return "", err
	}

	return lsp.DocumentURI("file://" + documentPath), nil
}

// GetPathFromURL returns the absolute path of the file which documenURI relates to
// it also checks if the file path is valid
func GetPathFromURL(uri lsp.DocumentURI, pathType string) (documentPath string, err error) {
	var pathFromURL string
	if IsURL(uri) {
		pathFromURL = strings.TrimPrefix(string(uri), "file://")
	} else {
		pathFromURL = string(uri)
	}

	absPath, err := filepath.Abs(pathFromURL)
	if err != nil {
		return "", err
	}

	core.FindRepoRoot()
	if strings.HasPrefix(absPath, core.RepoRoot) {
		pathType = strings.ToLower(pathType)
		switch pathType {
		case "file":
			if fs.FileExists(absPath) {
				return absPath, nil
			}
		case "path":
			if fs.PathExists(absPath) {
				return absPath, nil
			}
		default:
			return "", errors.New(fmt.Sprintf("invalid pathType %s, "+
				"can only be 'file' or 'path'", pathType))
		}
	}

	return "", errors.New(fmt.Sprintf("invalid path %s, path must be in repo root", absPath))
}
