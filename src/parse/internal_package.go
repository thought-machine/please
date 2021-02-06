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
  url = f"{{ .DownloadLocation }}/{{ .OS }}_{{ .Arch }}/{{ .PLZVersion }}/please_tools_{{ .PLZVersion }}.tar.xz",
)

genrule(
  name = "tools",
  srcs = [":download"],
  cmd = "tar -xf $SRC",
  outs = ["please_tools"],
  entry_points = {
    "lang_server": "please_tools/build_langserver",
    "jarcat": "please_tools/jarcat",
    "javac_worker": "please_tools/javac_worker",
    "junit_runner": "please_tools/junit_runner.jar",
    "please_go_filter": "please_tools/please_go_filter",
    "please_go_test": "please_tools/please_go_test",
    "please_go_install": "please_tools/please_go_install",
    "please_pex": "please_tools/please_pex",
    "please_sandbox": "please_tools/please_sandbox",
  },
  visibility = ["PUBLIC"],
  binary = True,
)

# TODO(jpoole): Delete this check in v16
if not CONFIG.FF_PLEASINGS:
  http_archive(
    name = "pleasings",
    strip_prefix = "pleasings-master",
    urls = ["https://github.com/thought-machine/pleasings/archive/master.zip"],
  )
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
	}{
		PLZVersion:       core.PleaseVersion.String(),
		OS:               runtime.GOOS,
		Arch:             runtime.GOARCH,
		DownloadLocation: config.Please.DownloadLocation.String(),
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
