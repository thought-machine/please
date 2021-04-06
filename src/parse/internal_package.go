package parse

import (
	"bytes"
	"runtime"
	"text/template"

	"github.com/thought-machine/please/src/core"
)

const InternalPackageName = "_please"

// TODO(jpoole): Make this the magic bindata thing once go 1.16 is out
// TODO(jpoole): make langserver configurable
const internalPackageTemplateStr = `
remote_file(
    name = "download",
    url = f"{{ .DownloadLocation }}/{{ .OS }}_{{ .Arch }}/{{ .PLZVersion }}/please_tools_{{ .PLZVersion }}.tar.xz",
)

# Can't use extract = True, as that would depend on :jarcat
genrule(
  name = "please_tools",
  srcs = [":download"],
  cmd = "tar -xf $SRC",
  outs = ["please_tools"],
)

{{ range $tool := .Tools }}
genrule(
    name = "{{$tool}}".removesuffix(".jar"),
    cmd = f"mv $SRC/{{$tool}} $OUT",
    srcs = [":please_tools"],
    outs = ["{{$tool}}"],
    visibility = ["PUBLIC"],
    binary = True,
)
{{ end }}

`

var internalPackageTemplate = template.New("_please")

func GetInternalPackage(config *core.Configuration) (string, error) {
	t, err := internalPackageTemplate.Parse(internalPackageTemplateStr)
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
		PLZVersion:       core.PleaseVersion.String(),
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
