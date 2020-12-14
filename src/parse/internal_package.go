package parse

import (
	"bytes"
	"github.com/thought-machine/please/src/core"
	"runtime"
	"text/template"
)

const InternalPackageName = "_please"

// TODO(jpoole): Make this the magic bindata thing once go 1.16 is out
// TODO(jpoole): make langserver configurable
const internalPackageTemplateStr = `
remote_file(
  name = "download",
  url = f"{{ .DownloadLocation }}/{{ .OS }}_{{ .Arch }}/{{ .PLZVersion }}/please_{{ .PLZVersion }}.tar.xz",
)

genrule(
  name = "tools",
  srcs = [":download"],
  cmd = "tar -xf $SRC",
  outs = ["please"],
  entry_points = {
    "lang_server": "please/build_langserver",
    "jarcat": "please/jarcat",
    "javac_worker": "please/javac_worker",
    "junit_runner": "please/junit_runner.jar",
    "please_go_filter": "please/please_go_filter",
    "please_go_test": "please/please_go_test",
    "please_pex": "please/please_pex",
    "please_sandbox": "please/please_sandbox",
  },
  visibility = ["PUBLIC"],
  binary = True,
)
`

var internalPackageTemplate = template.New("_please")


func GetInternalPackage(config *core.Configuration) (string, error) {
	t, err := internalPackageTemplate.Parse(internalPackageTemplateStr)
	if err != nil {
		return "", err
	}

	data := struct {
		PLZVersion string
		OS string
		Arch string
		DownloadLocation string
	}{
		PLZVersion: core.PleaseVersion.String(),
		OS: runtime.GOOS,
		Arch: runtime.GOARCH,
		DownloadLocation: config.Please.DownloadLocation.String(),
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}