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

	var arcatHash string
	switch fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH) {
	case "darwin_amd64":
		arcatHash = "0c674f9c324a5556d413fc9880fd9b4bd3df7e6f18f0c7258d7f3975c938e912"
	case "darwin_arm64":
		arcatHash = "aa359996e3e7c166c49421a14bf7ca80bc17508baf13556c4847383383d01b96"
	case "freebsd_amd64":
		arcatHash = "ee9abff21f102a11691a133a833143a31c724483fe3432fe875e4bf3731e9bd1"
	case "linux_amd64":
		arcatHash = "614a76d4e33526fe6cb75e98a70046661b5c6b7a0c3571c63af11dd525a02db3"
	case "linux_arm64":
		arcatHash = "dc50418e8962457dbcfc6e70463f088beef1f5b785c9fb030a14eeb67de0efa5"
	default:
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
