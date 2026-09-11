package parse

import (
	"bytes"
	_ "embed" // needed to use //go:embed
	"fmt"
	"runtime"
	"text/template"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
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

	arcatHash := publishedArcatHash()

	data := struct {
		ToolsURL  string
		Tools     []string
		ArcatHash string
		ExeSuffix string
	}{
		ToolsURL: url,
		Tools: []string{
			"build_langserver",
			"please_sandbox",
		},
		ArcatHash: arcatHash,
		ExeSuffix: fs.ExeSuffix,
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// publishedArcatHash returns the hash of the arcat release for the platform we are running on,
// or an empty string if there isn't one. An empty string leaves the arcat rule out of the
// internal package altogether rather than failing: everything else in there still works, and a
// user who points [build] arcattool at their own build never needs ours. See
// ArcatUnavailable for the warning that goes with it.
func publishedArcatHash() string {
	return arcatHashFor(fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH))
}

func arcatHashFor(platform string) string {
	switch platform {
	case "darwin_amd64":
		return "6af2cf108592535701aa9395f3a5deeb48a5dfbe8174a8ebe3d56bb93de2c255"
	case "darwin_arm64":
		return "5070ef05d14c66a85d438f400c6ff734a23833929775d6824b69207b704034bf"
	case "freebsd_amd64":
		return "05ad6ac45be3a4ca1238bb1bd09207a596f8ff5f885415f8df4ff2dc849fa04e"
	case "linux_amd64":
		return "aec85425355291e515cd10ac0addec3a5bc9e05c9d07af01aca8c34aaf0f1222"
	case "linux_arm64":
		return "8266cb95cc84b23642bca6567f8b4bd18de399c887cb5845ab6a901d0dba54d2"
	}
	return ""
}

// ArcatUnavailable reports whether the config still expects the arcat that Please would
// download, on a platform where there is no release to download. Nothing that needs arcat can
// work in that state - which includes extracting any plugin - so it is worth saying up front
// rather than letting it surface as a missing target much later.
func ArcatUnavailable(config *core.Configuration) bool {
	return publishedArcatHash() == "" && config.Build.ArcatTool == core.DefaultArcatTool
}
