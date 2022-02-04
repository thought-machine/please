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
	// Check for existing python fields first
	for _, section := range file.Sections {
		log.Warningf("%v", section.Name)
	}

	subsection := "python"
	section := "Plugin"
	file = ast.InjectField(file, "DefaultInterpreter", "python3", section, subsection, false)
	file = ast.InjectField(file, "PexTool", "@self//tools/please_pex:please_pex", section, subsection, false)
	file = ast.InjectField(file, "InterpreterOptions", "", section, subsection, false)
	file = ast.InjectField(file, "TestRunner", "unittest", section, subsection, false)
	file = ast.InjectField(file, "TestrunnerDeps", "//third_party/python:unittest_bootstrap", section, subsection, false)
	file = ast.InjectField(file, "Debugger", "pdb", section, subsection, false)
	file = ast.InjectField(file, "ModuleDir", "third_party.python", section, subsection, false)
	file = ast.InjectField(file, "PipTool", "", section, subsection, false)
	file = ast.InjectField(file, "DefaultPipRepo", "", section, subsection, false)
	file = ast.InjectField(file, "UsePypi", "true", section, subsection, false)
	file = ast.InjectField(file, "PipFlags", "", section, subsection, false)
	file = ast.InjectField(file, "DisableVendorFlags", "false", section, subsection, false)
	file = ast.InjectField(file, "WheelRepo", "true", section, subsection, false)
	file = ast.InjectField(file, "WheelNameScheme", "true", section, subsection, false)
	file = ast.InjectField(file, "WheelTool", "//tools/wheel_resolver", section, subsection, false)

	return file
}

func writeCCConfigFields(file ast.File) ast.File {
	subsection := "cc"
	section := "Plugin"
	file = ast.InjectField(file, "CCTool", "gcc", section, subsection, false)
	file = ast.InjectField(file, "CPPTool", "g++", section, subsection, false)
	file = ast.InjectField(file, "LDTool", "ld", section, subsection, false)
	file = ast.InjectField(file, "ARTool", "ar", section, subsection, false)
	file = ast.InjectField(file, "DefaultOptCFlags", "--std=c99 -O3 -pipe -DNDEBUG -Wall -Werror", section, subsection, false)
	file = ast.InjectField(file, "DefaultDbgCFlags", "--std=c99 -g3 -pipe -DDEBUG -Wall -Werror", section, subsection, false)
	file = ast.InjectField(file, "DefaultOptCppFlags", "--std=c++11 -O3 -pipe -DNDEBUG -Wall -Werror", section, subsection, false)
	file = ast.InjectField(file, "DefaultDbgCppFlags", "--std=c++11 -g3 -pipe -DDEBUG -Wall -Werror", section, subsection, false)
	file = ast.InjectField(file, "DefaultLdFlags", "-lpthread -ldl", section, subsection, false)
	file = ast.InjectField(file, "PkgConfigPath", "", section, subsection, false)
	file = ast.InjectField(file, "TestMain", "@self//unittest-pp:main", section, subsection, false)
	file = ast.InjectField(file, "DsymTool", "dsymutil", section, subsection, false)
	file = ast.InjectField(file, "AsmTool", "nasm", section, subsection, false)
	file = ast.InjectField(file, "DefaultNamespace", "", section, subsection, false)

	return file
}

func writeJavaConfigFields(file ast.File) ast.File {
	subsection := "java"
	section := "Plugin"
	file = ast.InjectField(file, "JavacTool", "javac", section, subsection, false)
	file = ast.InjectField(file, "JavacFlags", "", section, subsection, false)
	file = ast.InjectField(file, "JavacTestFlags", "", section, subsection, false)
	file = ast.InjectField(file, "JunitRunner", "//_please:junit_runner", section, subsection, false)
	file = ast.InjectField(file, "SourceLevel", "8", section, subsection, false)
	file = ast.InjectField(file, "TargetLevel", "8", section, subsection, false)
	file = ast.InjectField(file, "ReleaseLevel", "", section, subsection, false)
	file = ast.InjectField(file, "JavacWorker", "", section, subsection, false)
	file = ast.InjectField(file, "Toolchain", "//third_party/java:toolchain", section, subsection, false)
	file = ast.InjectField(file, "DefaultTestPackage", "true", section, subsection, false)
	file = ast.InjectField(file, "MavenRepo", "https://repo1.maven.org/maven2", section, subsection, true)
	file = ast.InjectField(file, "MavenRepo", "https://jcenter.bintray.com", section, subsection, true)
	return file
}

func createTarget(location string, plugin string) error {
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
