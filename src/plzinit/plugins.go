package plzinit

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/please-build/gcfg/ast"
	"github.com/thought-machine/please/src/core"
)

const pluginRepoTemplate = `
plugin_repo(
  name = "%s",
  revision = "v0.0.0",
  plugin = "%s-rules",
  owner = "please-build",
)
`

// InitPlugin initialises a plugin
func InitPlugins(plugins []string) {
	configExists := core.FindRepoRoot()
	if !configExists {
		log.Warning("No config found. Creating one in this directory.")
		InitConfig(".", false, true)
	}

	log.Warningf("Initialising plugin(s):")
	for _, p := range plugins {
		log.Warningf("%v", p)
		initPlugin(p)
	}
}

func initPlugin(plugin string) {
	log.Warningf("Found existing config. Just leave existing fields and inject new fields")
	injectPluginConfig(plugin)
	createTarget("plugins/BUILD", plugin)
}

func injectPluginConfig(plugin string) error {
	configPath := path.Join(core.RepoRoot, ".plzconfig")
	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		log.Warningf("config file %s does not exist", configPath)
		return err
	}

	configFile, err := os.Open(configPath)
	if err != nil {
		panic(err)
	}
	defer configFile.Close()
	file := ast.Read(configFile)
	switch plugin {
	case "python":
		log.Warningf("Write python config fields")
		file = writePythonConfigFields(file)
	case "java":
		log.Warningf("Write java config fields")
	case "cc":
		log.Warningf("Write cc config fields")
	default:
		log.Fatalf("Failed to initialise unrecognised plugin \"%s\"", plugin)
	}
	ast.Write(file, configPath)
	return nil
}

func writePythonConfigFields(file ast.File) ast.File {
	plugin := "python"
	file = ast.InjectField(file, "DefaultInterpreter", "python3", "Plugin", plugin, false)
	file = ast.InjectField(file, "PexTool", "bar", "Plugin", plugin, false)
	file = ast.InjectField(file, "ModuleDir", "third_party.python", "Plugin", plugin, false)
	file = ast.InjectField(file, "UsePypi ", "false", "Plugin", plugin, false)

	return file
}

func writeCCConfigFields(file ast.File) ast.File {
	plugin := "cc"
	file = ast.InjectField(file, "CCTool", "gcc", "Plugin", plugin, false)
	file = ast.InjectField(file, "CPPTool", "g++", "Plugin", plugin, false)
	file = ast.InjectField(file, "LDTool", "ld", "Plugin", plugin, false)
	file = ast.InjectField(file, "ARTool", "ar", "Plugin", plugin, false)
	file = ast.InjectField(file, "DefaultOptCFlags", "--std=c99 -O3 -pipe -DNDEBUG -Wall -Werror", "Plugin", plugin, false)
	file = ast.InjectField(file, "DefaultDbgCFlags", "--std=c99 -g3 -pipe -DDEBUG -Wall -Werror", "Plugin", plugin, false)
	file = ast.InjectField(file, "DefaultOptCppFlags", "--std=c++11 -O3 -pipe -DNDEBUG -Wall -Werror", "Plugin", plugin, false)
	file = ast.InjectField(file, "DefaultDbgCppFlags", "--std=c++11 -g3 -pipe -DDEBUG -Wall -Werror", "Plugin", plugin, false)
	file = ast.InjectField(file, "DefaultLdFlags", "-lpthread -ldl", "Plugin", plugin, false)
	file = ast.InjectField(file, "PkgConfigPath", "", "Plugin", plugin, false)
	file = ast.InjectField(file, "TestMain", "@self//unittest-pp:main", "Plugin", plugin, false)
	file = ast.InjectField(file, "DsymTool", "dsymutil", "Plugin", plugin, false)
	file = ast.InjectField(file, "AsmTool", "nasm", "Plugin", plugin, false)
	file = ast.InjectField(file, "DefaultNamespace", "", "Plugin", plugin, false)

	return file
}

func createTarget(location string, plugin string) error {
	dir := filepath.Dir("plugins/")
	if err := os.MkdirAll(dir, core.DirPermissions); err != nil {
		log.Fatalf("failed to create plugins directory: %v", err)
	}

	f, err := os.OpenFile(location, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, pluginRepoTemplate, plugin, plugin)

	return err
}
