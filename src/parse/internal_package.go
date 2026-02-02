package parse

import (
	"bytes"
	_ "embed" // needed to use //go:embed
	"fmt"
	"runtime"
	"text/template"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/version"
)

const InternalPackageName = "_please"

//go:embed internal.tmpl
var internalPackageTemplateStr string

func GetInternalPackage(config *core.Configuration) (string, error) {
	t, err := template.New("_please").Parse(internalPackageTemplateStr)
	if err != nil {
		return "", err
	}

	url := config.Please.ToolsURL.String()
	if url == "" {
		url = fmt.Sprintf("%s/%s_%s/%s/please_tools_%s.tar.xz", config.Please.DownloadLocation, runtime.GOOS, runtime.GOARCH, version.PleaseVersion, version.PleaseVersion)
	}

	arcatHash := ""
	switch fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH) {
	case "darwin_amd64":
		arcatHash = "6af2cf108592535701aa9395f3a5deeb48a5dfbe8174a8ebe3d56bb93de2c255"
	case "darwin_arm64":
		arcatHash = "5070ef05d14c66a85d438f400c6ff734a23833929775d6824b69207b704034bf"
	case "freebsd_amd64":
		arcatHash = "05ad6ac45be3a4ca1238bb1bd09207a596f8ff5f885415f8df4ff2dc849fa04e"
	case "linux_amd64":
		arcatHash = "aec85425355291e515cd10ac0addec3a5bc9e05c9d07af01aca8c34aaf0f1222"
	case "linux_arm64":
		arcatHash = "8266cb95cc84b23642bca6567f8b4bd18de399c887cb5845ab6a901d0dba54d2"
	}

	if arcatHash == "" {
		return "", fmt.Errorf("arcat tool not supported for platform: %s_%s", runtime.GOOS, runtime.GOARCH)
	}

	data := struct {
		ToolsURL  string
		Tools     []string
		ArcatHash string
	}{
		ToolsURL: url,
		Tools: []string{
			"build_langserver",
			"please_sandbox",
		},
		ArcatHash: arcatHash,
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
