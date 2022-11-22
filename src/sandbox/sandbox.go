package sandbox

import (
	"embed"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/thought-machine/please/src/cli/logging"
)

var log = logging.Log

//go:embed cmd/sandbox cmd/sandbox.sha256
var f embed.FS

var toolPath = ""

var unpack sync.Once

func unpackSandboxTool() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	hash, err := f.ReadFile("cmd/sandbox.sha256")
	if err != nil {
		return "", err
	}

	toolPath := filepath.Join(wd, "plz-out/sandbox_tools", strings.TrimSpace(strings.Split(string(hash), " ")[0]))
	if _, err := os.Lstat(toolPath); err == nil {
		return toolPath, nil
	}

	sandbox, err := f.Open("cmd/sandbox")
	if err != nil {
		return "", err
	}
	defer sandbox.Close()

	if err := os.MkdirAll("plz-out/sandbox_tools", os.ModeDir|0775); err != nil {
		return "", err
	}

	to, err := os.OpenFile(toolPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0776)
	if err != nil {
		return "", err
	}
	defer to.Close()

	if _, err := io.Copy(to, sandbox); err != nil {
		return "", err
	}

	return toolPath, nil
}

// Tool unpacks the sandbox tool (if needed) and returns the path
func Tool() string {
	unpack.Do(func() {
		path, err := unpackSandboxTool()
		if err != nil {
			log.Fatalf("Failed to unpack sandbox tool: %v", err)
		}

		toolPath = path
	})
	return toolPath
}
