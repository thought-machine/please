package plzinit

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/please-build/buildtools/build"
	"github.com/please-build/gcfg/ast"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

const pluginRepoTemplate = `plugin_repo(
  name = "%s",
  revision = "%s",
  plugin = "%s-rules",
  owner = "please-build",
)
`

var pluginInitFns = map[string]func() (map[string]string, error){
	"go": initGo,
}

// Mapping of the built in config from Please v16 to the new plugins introduced in v17
type v16Mapping = map[string]string

var pluginVersion16Map = map[string]v16Mapping{
	"go": {
		"gotool":           "GoTool",
		"importpath":       "ImportPath",
		"cgocctool":        "CCTool",
		"cgoenabled":       "CGoEnabled",
		"pleasegotool":     "PleaseGoTool",
		"delvetool":        "DelveTool",
		"defaultstatic":    "DefaultStatic",
		"gotestrootcompat": "TestRootCompat",
		"cflags":           "CFlags",
		"ldflags":          "LdFlags",
	},
	"python": {
		"piptool":             "PipTool",
		"pipflags":            "PipFlags",
		"pextool":             "PexTool",
		"defaultinterpreter":  "DefaultInterpreter",
		"testrunner":          "TestRunner",
		"debugger":            "Debugger",
		"moduledir":           "ModuleDir",
		"defaultpiprepo":      "DefaultPipRepo",
		"wheelrepo":           "WheelRepo",
		"wheelnamescheme":     "WheelNameScheme",
		"interpreteroptions":  "InterpreterOptions",
		"disablevendorflags":  "DisableVendorFlags",
		"usepypi":             "UsePypi",
		"testrunnerbootstrap": "TestrunnerDeps",
	},
	"cc": {
		"cctool":             "CCTool",
		"cpptool":            "CPPTool",
		"ldtool":             "LDTool",
		"artool":             "ARTool",
		"defaultoptcflags":   "DefaultOptCFlags",
		"defaultdbgcflags":   "DefaultDbgCFlags",
		"defaultoptcppflags": "DefaultOptCppFlags",
		"defaultdbgcppflags": "DefaultDbgCppFlags",
		"defaultldflags":     "DefaultLdFlags",
		"pkgconfigpath":      "PkgConfigPath",
		"testmain":           "TestMain",
		"dsymtool":           "DsymTool",
	},
	"java": {
		"javactool":          "JavacTool",
		"javacworker":        "JavacWorker",
		"junitrunner":        "JunitRunner",
		"defaulttestpackage": "DefaultTestPackage",
		"releaselevel":       "ReleaseLevel",
		"targetlevel":        "TargetLevel",
		"javacflags":         "JavacFlags",
		"javactestflags":     "JavacTestFlags",
		"defaultmavenrepo":   "MavenRepo",
		"toolchain":          "Toolchain",
	},
}

func info(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

// InitPlugins initialises one or more plugins by inserting plugin config values into
// the host repo config file, and creating a build target in //plugins.
func InitPlugins(plugins []string, version string) error {
	log.Debug("Initialising plugin(s): %v", plugins)

	// Check that we're in a plz repo
	configPath := filepath.Join(core.RepoRoot, ".plzconfig")
	if !fs.FileExists(configPath) {
		return fmt.Errorf("You don't appear to be in a plz repo.")
	}

	configFile, err := os.Open(configPath)
	if err != nil {
		return fmt.Errorf("Failed to open plz config file")
	}
	defer configFile.Close()

	// Read config file into AST
	file := ast.Read(configFile)

	for _, p := range plugins {
		config, err := initPlugin(p, version)
		if err != nil {
			return fmt.Errorf("Could not initialise plugin %s. Got error: %s", p, err)
		}
		file = writeFieldsToConfig(p, file, config)
	}

	return ast.Write(file, configPath)
}

// initPlugin initialises the plugin, performing any plugin specific operations, returning the plugin config
func initPlugin(plugin, version string) (map[string]string, error) {
	if err := createPluginTarget("plugins/BUILD", plugin, version); err != nil {
		return nil, err
	}

	if fn, ok := pluginInitFns[plugin]; ok {
		return fn()
	}
	return nil, nil
}

func writeFieldsToConfig(plugin string, plzConfig ast.File, pluginConfig map[string]string) ast.File {
	section := "Plugin"

	pluginName := strings.ReplaceAll(plugin, "-", "_")

	// Check for existing plugin section
	if s := plzConfig.MaybeGetSection(section, pluginName); s != nil {
		info("Plugin config section already exists, so init did nothing.")
		return plzConfig
	}

	// Inject the preloadsubincludes
	// TODO(sam): We can get the actual name of the package containing the build_defs
	// if we build the plugin target, which we do below. Refactor this to build the target
	// earlier and use the build_defs dir specified in the plugin config
	subincludeStr := "///" + pluginName + "//build_defs:" + pluginName
	plzConfig = ast.InjectField(plzConfig, "preloadsubincludes", subincludeStr, "parse", "", true)

	// Write plugin target value
	plzConfig = ast.InjectField(plzConfig, "Target", "//plugins:"+pluginName, section, pluginName, false)

	for k, v := range pluginConfig {
		plzConfig = ast.InjectField(plzConfig, k, v, section, pluginName, false)
	}

	// Migrate any existing language fields to their plugin equivalents
	if configMap, ok := pluginVersion16Map[plugin]; ok {
		for _, s := range plzConfig.Sections {
			if s.Key == pluginName {
				for _, field := range s.Fields {
					if plugVal, ok := configMap[strings.ToLower(field.Name)]; ok {
						plzConfig = ast.InjectField(plzConfig, plugVal, field.Value, section, pluginName, true)
					}
				}
			}
		}
	}

	return plzConfig
}

// targetExistsInFile checks to see if the plugin target already exists
// in plugins/BUILD
func targetExistsInFile(location, target string) (bool, error) {
	if !fs.FileExists(location) {
		return false, nil
	}

	b, err := os.ReadFile(location)
	if err != nil {
		return false, err
	}

	f, err := build.Parse(location, b)
	if err != nil {
		return false, err
	}

	for _, rule := range f.Rules("plugin_repo") {
		if rule.Name() == target {
			return true, nil
		}
	}
	return false, nil
}

// createPluginTarget writes the plugin target to plugins/BUILD
func createPluginTarget(location, plugin, version string) error {
	pluginTarget := strings.ReplaceAll(plugin, "-", "_")
	exists, err := targetExistsInFile(location, pluginTarget)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	pkg := filepath.Dir(location)
	if err := os.MkdirAll(pkg, core.DirPermissions); err != nil {
		return err
	}

	f, err := os.OpenFile(location, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if version == "" {
		revision, err := getLatestRevision(plugin)
		if err != nil {
			return err
		}
		version = revision
	}
	_, err = fmt.Fprintf(f, pluginRepoTemplate, pluginTarget, version, plugin)

	return err
}

type Response []struct {
	Name       string `json:"name"`
	ZipballURL string `json:"zipball_url"`
	TarballURL string `json:"tarball_url"`
	Commit     struct {
		Sha string `json:"sha"`
		URL string `json:"url"`
	} `json:"commit"`
	NodeID string `json:"node_id"`
}

// getLatestRevision pulls the latest release tag for the plugin from github
func getLatestRevision(plugin string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/please-build/%s-rules/tags", plugin)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("accept", "application/vnd.github.v3+json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	} else if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Failed to download plugin: %s %s", resp.Status, string(body))
	}

	var result Response
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return result[0].Name, nil
}
