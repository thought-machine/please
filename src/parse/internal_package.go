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

	data := struct {
		ToolsURL string
		Tools    []string
	}{
		ToolsURL: url,
		Tools: []string{
			"build_langserver",
			"please_sandbox",
		},
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
