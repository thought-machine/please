package plzinit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/please-build/buildtools/build"

	"github.com/thought-machine/please/src/fs"
)

const buildFilePath = "third_party/go/BUILD"

type goVersionResp = []struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

// getLatestGoVersion fetches the latest stable Go version from the Go website
func getLatestGoVersion() (string, error) {
	resp, err := http.Get("https://golang.org/dl/?mode=json")
	if err != nil {
		return "", fmt.Errorf("failed to fetch Go versions: %w", err)
	}
	defer resp.Body.Close()

	var versions = make(goVersionResp, 0, 1)
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return "", fmt.Errorf("failed to decode JSON response: %w", err)
	}

	// Filter only stable versions and sort them
	for _, v := range versions {
		if v.Stable {
			// Assume the response is ordered. It seems to always return just the latest version anyway.
			return v.Version, nil
		}
	}

	return runtime.Version(), nil
}

func parseBuildFile() (*build.File, error) {
	bs, _ := os.ReadFile(buildFilePath)
	return build.Parse(buildFilePath, bs)
}

func saveFile(buildFile *build.File) error {
	bs := build.Format(buildFile)
	if err := fs.EnsureDir(buildFilePath); err != nil {
		return err
	}

	return os.WriteFile(buildFilePath, bs, 0666)
}

func initGo() (map[string]string, error) {
	goVer, err := getLatestGoVersion()
	if err != nil {
		return nil, err
	}
	goVer = strings.TrimPrefix(goVer, "go")

	if err := fs.EnsureDir("third_party/go/BUILD"); err != nil {
		return nil, err
	}

	buildFile, err := parseBuildFile()
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", buildFilePath, err)
	}

	toolchainRule := "//third_party/go:toolchain|go"
	if rules := buildFile.Rules("go_toolchain"); len(rules) == 1 {
		toolchainRule = fmt.Sprintf("//third_party/go:%v|go", rules[0].Name())
	} else if !hasRule(rules, "toolchain") {
		buildFile.Stmt = append(buildFile.Stmt, goToolchain("toolchain", goVer))
	}

	stdRule := "//third_party/go:std"
	if rules := buildFile.Rules("go_stdlib"); len(rules) == 1 {
		toolchainRule = fmt.Sprintf("//third_party/go:%v", rules[0].Name())
	} else if !hasRule(rules, "std") {
		buildFile.Stmt = append(buildFile.Stmt, stdLib("std"))
	}

	if err := saveFile(buildFile); err != nil {
		return nil, err
	}

	return map[string]string{
		"GoTool": toolchainRule,
		"STDLib": stdRule,
	}, nil
}

func goToolchain(name, version string) *build.CallExpr {
	r := build.NewRule(&build.CallExpr{})
	r.SetKind("go_toolchain")
	r.SetAttr("name", &build.StringExpr{Value: name})
	r.SetAttr("version", &build.StringExpr{Value: version})
	return r.Call
}

func stdLib(name string) *build.CallExpr {
	r := build.NewRule(&build.CallExpr{})
	r.SetKind("go_stdlib")
	r.SetAttr("name", &build.StringExpr{Value: name})
	return r.Call
}

func hasRule(rules []*build.Rule, name string) bool {
	for _, r := range rules {
		if r.Name() == name {
			return true
		}
	}
	return false
}
