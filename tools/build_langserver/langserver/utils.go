package langserver

import (
	"bufio"
	"context"
	"core"
	"fmt"
	"fs"
	"os"
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
			return "", fmt.Errorf(fmt.Sprintf("invalid pathType %s, "+
				"can only be 'file' or 'path'", pathType))
		}
	}

	return "", fmt.Errorf(fmt.Sprintf("invalid path %s, path must be in repo root", absPath))
}

// ReadFile takes a DocumentURI and reads the file into a slice of string
func ReadFile(ctx context.Context, uri lsp.DocumentURI) ([]string, error) {
	path, err := GetPathFromURL(uri, "file")
	if err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			log.Info("process cancelled.")
			return nil, nil
		default:
			lines = append(lines, scanner.Text())
		}
	}

	return lines, scanner.Err()
}
