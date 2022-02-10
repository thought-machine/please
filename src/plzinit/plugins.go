package plzinit

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/please-build/gcfg/ast"
	"github.com/thought-machine/please/src/core"
)

const pluginRepoTemplate = `
plugin_repo(
  name = "%s",
  revision = "%s",
  plugin = "%s-rules",
  owner = "please-build",
)
`

// InitPlugins initialises one or more plugins by inserting plugin config values into
// the host repo config file, and creating a build target in //plugins.
func InitPlugins(plugins []string) {
	configExists := core.FindRepoRoot()
	if !configExists {
		log.Warning("No config found. Creating one in this directory.")
		InitConfig(".", false, true)
	}

	log.Debug("Initialising plugin(s): %v", plugins)
	for _, p := range plugins {
		if err := initPlugin(p); err != nil {
			log.Warningf("Could not initialise plugin %s. Got error: %v", p, err)
		}
	}
}

func initPlugin(plugin string) error {
	log.Warningf("Inserting plugin config values into .plzconfig")
	if err := injectPluginConfig(plugin); err != nil {
		return err
	}
	if err := createTarget("plugins/BUILD", plugin); err != nil {
		return err
	}
	return nil
}

func injectPluginConfig(plugin string) error {
	configPath := path.Join(core.RepoRoot, ".plzconfig")
	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		log.Warningf("config file %s does not exist", configPath)
		return err
	}

	configFile, err := os.Open(configPath)
	if err != nil {
		return err
	}
	defer configFile.Close()
	file := ast.Read(configFile)
	switch plugin {
	case "python":
		file = writePythonConfigFields(file)
	case "java":
		file = writeJavaConfigFields(file)
	case "cc":
		file = writeCCConfigFields(file)
	default:
		log.Fatalf("Failed to initialise plugin. \"%s\" not recognised", plugin)
	}
	ast.Write(file, configPath)
	return nil
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

	section := "Plugin"
	subsection := "python"

	// Check for existing python fields first
	for _, s := range file.Sections {
		if s.Key == section {
			for _, field := range s.Fields {
				log.Warningf("%v\t%v", field.Name, field.Value)
				if plugVal, ok := configMap[strings.ToLower(field.Name)]; ok {
					log.Warningf("Got a hit with %v", field.Name)
					file = ast.InjectField(file, plugVal, field.Value, section, subsection, false)
				}
			}
		}
	}

	return file
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

	// in main plz only
	// LinkWithLdTool
	// // Coverage
	// // ClangModules

	// in cc plugin only
	// AsmTool
	// DefaultNamespace

	subsection := "cc"
	section := "Plugin"

	// Check for existing cc fields first
	for _, s := range file.Sections {
		if s.Key == subsection {
			for _, field := range s.Fields {
				log.Warningf("%v\t%v", field.Name, field.Value)
				if plugVal, ok := configMap[strings.ToLower(field.Name)]; ok {
					log.Warningf("Got a hit with %v", field.Name)
					file = ast.InjectField(file, plugVal, field.Value, section, subsection, false)
				}
			}
		}
	}

	return file
}

func writeJavaConfigFields(file ast.File) ast.File {

	// in main but not in plugin
	// JlinkTool
	// JavaHome
	// JarCatTool
	// SourceLevel

	// Main plz
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

	subsection := "java"
	section := "Plugin"

	// Check for existing java fields first
	for _, s := range file.Sections {
		if s.Key == subsection {
			for _, field := range s.Fields {
				log.Warningf("%v\t%v", field.Name, field.Value)
				if plugVal, ok := configMap[strings.ToLower(field.Name)]; ok {
					log.Warningf("Got a hit with %v", field.Name)
					file = ast.InjectField(file, plugVal, field.Value, section, subsection, false)
				}
			}
		}
	}

	return file
}

func targetExistsInFile(location, plugin string) bool {
	b, err := ioutil.ReadFile(location)
	if err != nil {
		panic(err)
	}

	str := `plugin_repo(
  name = "` + plugin
	exists, err := regexp.Match(str, b)
	if err != nil {
		panic(err)
	}
	return exists
}

func createTarget(location, plugin string) error {
	if targetExistsInFile(location, plugin) {
		return nil
	}

	dir := filepath.Dir("plugins/")
	if err := os.MkdirAll(dir, core.DirPermissions); err != nil {
		return err
	}

	f, err := os.OpenFile(location, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	revision, err := getLatestRevision(plugin)
	if err != nil {
		return err
	}
	log.Warningf("Got revision %s", revision)
	_, err = fmt.Fprintf(f, pluginRepoTemplate, plugin, revision, plugin)

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

func getLatestRevision(plugin string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/please-build/%s-rules/tags", plugin)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("accept", "application/vnd.github.v3+json")
	client := &http.Client{}
	resp, err := client.Do(req)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("failed to get latest release of plugin")
	}

	var result Response
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return result[0].Name, err
}
