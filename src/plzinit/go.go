package plzinit

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
)

const goConfig = `
[go]
;gotool = ...
;goroot - ...
`

// golangConfig prompts the user and returns the go config to add to the .plzconfig
func golangConfig(dir string, noPrompt bool) string {
	goModule, moduleFound := findGoModule(dir)
	config := goConfig

	if noPrompt {
		if moduleFound {
			return config + fmt.Sprintf("importpath = %s\n", goModule)
		}
		return ""
	}

	if !moduleFound {
		return ""
	}

	goOnPath, _ := core.LookPath("go", core.DefaultPath)
	if goOnPath == "" {
		fmt.Println("Warning: go is not found on the default path. Please configure gotool, or goroot under [go] in your .plzconfig")
		fmt.Println("Please doesn't use the host machines $PATH variable. Instead the path is configured in .plzconfig with a default of")
		fmt.Println("\n[build]")
		fmt.Println("path=", strings.Join(core.DefaultPath, ":"))
		fmt.Println("\nYou may also add {GOROOT}/go/bin to the path if you prefer")
	}

	var importPath string
	var err error
	if moduleFound {
		importPath = goModule
	} else {
		importPath, err = cli.Prompt("Import path (i.e. go module)", "")
	}
	if err != nil {
		os.Exit(1)
		return ""
	}
	if importPath != "" {
		if !moduleFound {
			fmt.Sprintln("You may also want to `go mod init " + importPath + "` for better IDE integration")
		}
		config += fmt.Sprintf("importpath = %s\n", importPath)
	} else {
		config += fmt.Sprintln(";importpath = ...")
	}

	return config
}

func findGoModule(dir string) (string, bool) {
	goModFile, err := os.Open(filepath.Join(dir, "go.mod"))
	if err != nil {
		return "", false
	}
	defer goModFile.Close()

	scanner := bufio.NewScanner(goModFile)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "module ") {
			return strings.TrimPrefix(line, "module "), true
		}
	}

	return "", false
}
