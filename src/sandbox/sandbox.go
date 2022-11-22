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

// SandboxTool unpacks the sandbox tool (if needed) and returns the path
func SandboxTool() string {
	unpack.Do(func() {
		wd, _ := os.Getwd()
		hash, err := f.ReadFile("cmd/sandbox.sha256")
		if err != nil {
			log.Fatalf("Failed to unpack sandbox tool: %v", err)
		}
		toolPath = filepath.Join(wd, "plz-out/sandbox_tools", strings.TrimSpace(strings.Split(string(hash), " ")[0]))
		if _, err := os.Lstat(toolPath); err == nil {
			return
		}
		sandbox, err := f.Open("cmd/sandbox")
		if err != nil {
			log.Fatalf("Failed to load embedded sandbox tool: %v", err)
		}
		defer sandbox.Close()

		os.MkdirAll("plz-out/sandbox_tools", os.ModeDir|0775)
		to, err := os.OpenFile(toolPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 777)
		if err != nil {
			log.Fatalf("Failed to create sandbox tool %v: %v", toolPath, err)
		}
		defer to.Close()

		if _, err := io.Copy(to, sandbox); err != nil {
			log.Fatalf("Failed to udnpack sandbox tool: %v", err)
		}

	})
	return toolPath
}
