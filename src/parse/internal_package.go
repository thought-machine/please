package parse

import (
	"bytes"
	_ "embed" // needed to use //go:embed
	"runtime"
	"text/template"

	"github.com/thought-machine/please/src/core"
)

const InternalPackageName = "_please"

// TODO(jpoole): make langserver configurable
//go:embed internal.tmpl
var internalPackageTemplateStr string

func GetInternalPackage(config *core.Configuration) (string, error) {
	t, err := template.New("_please").Parse(internalPackageTemplateStr)
	if err != nil {
		return "", err
	}

	data := struct {
		PLZVersion       string
		OS               string
		Arch             string
		DownloadLocation string
		Tools            []string
	}{
		PLZVersion:       core.PleaseVersion,
		OS:               runtime.GOOS,
		Arch:             runtime.GOARCH,
		DownloadLocation: config.Please.DownloadLocation.String(),
		Tools: []string{
			"build_langserver",
			"jarcat",
			"javac_worker",
			"junit_runner.jar",
			"please_go",
			"please_go_embed",
			"please_go_filter",
			"please_pex",
			"please_sandbox",
		},
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
