package plzinit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/please-build/buildtools/build"

	"github.com/thought-machine/please/src/fs"
)

const (
	buildFilePath     = "third_party/go/BUILD"
	rootBuildFileName = "BUILD"
	goModFileName     = "go.mod"
	goModFilegroup    = "gomod"
)

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

func parseBuildFile(path string) (*build.File, error) {
	bs, _ := os.ReadFile(path)
	return build.Parse(path, bs)
}

func saveFile(buildFile *build.File, path string) error {
	bs := build.Format(buildFile)
	if err := fs.EnsureDir(path); err != nil {
		return err
	}

	return os.WriteFile(path, bs, 0666)
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

	buildFile, err := parseBuildFile(buildFilePath)
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

	if err := saveFile(buildFile, buildFilePath); err != nil {
		return nil, err
	}

	config := map[string]string{
		"GoTool": toolchainRule,
		"STDLib": stdRule,
	}

	// Point the plugin at the repo's go.mod, if it has one. Without this, tools like puku
	// can't reconcile the module requirements against the third party build file.
	modFile, err := initGoMod(".")
	if err != nil {
		return nil, err
	}
	if modFile != "" {
		config["ModFile"] = modFile
	}

	return config, nil
}

// initGoMod exports the repo's go.mod via a filegroup in the root build file, returning the
// label of that filegroup. It returns an empty label if the repo doesn't have a go.mod.
func initGoMod(dir string) (string, error) {
	if _, ok := findGoModule(dir); !ok {
		return "", nil
	}

	path := filepath.Join(dir, rootBuildFileName)
	buildFile, err := parseBuildFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to parse %s: %w", path, err)
	}

	// Reuse an existing filegroup if the repo already exports go.mod under another name.
	for _, rule := range buildFile.Rules("filegroup") {
		for _, src := range rule.AttrStrings("srcs") {
			if src == goModFileName {
				return "//:" + rule.Name(), nil
			}
		}
	}

	buildFile.Stmt = append(buildFile.Stmt, modFilegroup(goModFilegroup))
	if err := saveFile(buildFile, path); err != nil {
		return "", err
	}

	return "//:" + goModFilegroup, nil
}

func modFilegroup(name string) *build.CallExpr {
	r := build.NewRule(&build.CallExpr{})
	r.SetKind("filegroup")
	r.SetAttr("name", &build.StringExpr{Value: name})
	r.SetAttr("srcs", &build.ListExpr{List: []build.Expr{&build.StringExpr{Value: goModFileName}}})
	r.SetAttr("visibility", &build.ListExpr{List: []build.Expr{&build.StringExpr{Value: "PUBLIC"}}})
	return r.Call
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
