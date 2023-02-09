package plzinit

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

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
		file, err = initPlugin(p, version, file)
		if err != nil {
			return fmt.Errorf("Could not initialise plugin %s. Got error: %s", p, err)
		}
	}

	ast.Write(file, configPath)
	return nil
}

func initPlugin(plugin, version string, file ast.File) (ast.File, error) {
	if err := createTarget("plugins/BUILD", plugin, version); err != nil {
		return file, err
	}

	return injectPluginConfig(plugin, file), nil
}

func injectPluginConfig(plugin string, file ast.File) ast.File {
	switch plugin {
	case "python":
		return writePythonConfigFields(file)
	case "java":
		return writeJavaConfigFields(file)
	case "cc":
		return writeCCConfigFields(file)
	case "go":
		return writeGoConfigFields(file)
	default:
		return writeFieldsToConfig(plugin, file, nil)
	}
}

func writePythonConfigFields(file ast.File) ast.File {
	configMap := map[string]string{
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
	}

	return writeFieldsToConfig("python", file, configMap)
}

func writeCCConfigFields(file ast.File) ast.File {
	configMap := map[string]string{
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
	}

	return writeFieldsToConfig("cc", file, configMap)
}

func writeJavaConfigFields(file ast.File) ast.File {
	configMap := map[string]string{
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
	}

	return writeFieldsToConfig("java", file, configMap)
}

func writeGoConfigFields(file ast.File) ast.File {
	configMap := map[string]string{
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
	}

	return writeFieldsToConfig("go", file, configMap)
}

func writeFieldsToConfig(plugin string, file ast.File, configMap map[string]string) ast.File {
	section := "Plugin"

	pluginName := strings.ReplaceAll(plugin, "-", "_")

	// Check for existing plugin section
	if s := file.MaybeGetSection(section, pluginName); s != nil {
		info("Plugin config section already exists, so init did nothing.")
		return file
	}

	// Inject the preloadsubincludes
	// TODO(sam): We can get the actual name of the package containing the build_defs
	// if we build the plugin target, which we do below. Refactor this to build the target
	// earlier and use the build_defs dir specified in the plugin config
	subincludeStr := "///" + pluginName + "//build_defs:" + pluginName
	file = ast.InjectField(file, "preloadsubincludes", subincludeStr, "parse", "", true)

	// Write plugin target value
	file = ast.InjectField(file, "Target", "//plugins:"+pluginName, section, pluginName, false)

	// Migrate any existing language fields to their plugin equivalents
	if configMap != nil {
		for _, s := range file.Sections {
			if s.Key == pluginName {
				for _, field := range s.Fields {
					if plugVal, ok := configMap[strings.ToLower(field.Name)]; ok {
						file = ast.InjectField(file, plugVal, field.Value, section, pluginName, true)
					}
				}
			}
		}
	}

	return file
}

// targetExistsInFile checks to see if the plugin target already exists
// in plugins/BUILD
func targetExistsInFile(location, target string) bool {
	if !fs.FileExists(location) {
		return false
	}

	b, err := os.ReadFile(location)
	if err != nil {
		panic(err)
	}

	//TODO: Might want to pull in the state object here one day so we can query the build
	// graph instead of using regexp

	str := "plugin_repo\\(.+name = \"" + target + "\""
	exists, err := regexp.Match("(?s)"+str, b)
	if err != nil {
		panic(err)
	}
	return exists
}

// createTarget writes the plugin target to plugins/BUILD
func createTarget(location, plugin, version string) error {
	pluginTarget := strings.ReplaceAll(plugin, "-", "_")
	if targetExistsInFile(location, pluginTarget) {
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
