package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/thought-machine/please/docs/tools/lexicon_templater/rules"
	"github.com/thought-machine/please/docs/tools/plugin_config_tool/plugin"
	"github.com/thought-machine/please/src/core"

	"github.com/peterebden/go-cli-init/v5/flags"
	"github.com/please-build/gcfg"
)

// formatConfigKey converts the config key from snake_case to CamelCase
func formatConfigKey(key string) string {
	key = strings.ToLower(key)
	parts := strings.Split(key, "_")
	for i, p := range parts {
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "")
}

var opts struct {
	Plugin string `long:"plugin_dir" required:"true" description:"Path to the plugin"`
	Please string `long:"plz" env:"DATA_PLEASE" required:"true" description:"Path to the Please binary"`
}

func getConfigFields(config *core.Configuration) []*plugin.ConfigField {
	fields := make([]*plugin.ConfigField, 0, len(config.PluginConfig))
	for name, field := range config.PluginConfig {
		key := field.ConfigKey
		if key == "" {
			key = formatConfigKey(name)
		}

		f := &plugin.ConfigField{
			Name:       key,
			Type:       field.Type,
			Help:       field.Help,
			Inherit:    field.Inherit,
			Repeatable: field.Repeatable,
			Optional:   field.Optional,
		}
		if len(field.DefaultValue) == 1 {
			f.Defaults = true
			f.DefaultValue = field.DefaultValue[0]
		} else if len(field.DefaultValue) > 1 {
			f.Defaults = true
			f.DefaultValue = fmt.Sprintf("[%v]", field.DefaultValue)
		}

		if f.Type == "" {
			f.Type = "string"
		}
		fields = append(fields, f)

	}
	return fields
}

func getRules(buildDefsDirs []string) *rules.Rules {
	if len(buildDefsDirs) == 0 {
		buildDefsDirs = []string{"build_defs"}
	}

	var defs []string
	for _, dir := range buildDefsDirs {
		err := filepath.WalkDir(dir, func(path string, de fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !de.IsDir() && filepath.Ext(path) == ".build_defs" {
				defs = append(defs, path)
			}
			return nil
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	}

	args := []string{"--noupdate", "query", "rules"}
	cmd := exec.Command(opts.Please, append(args, defs...)...)
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "faild to get rules, err: %v\n", errb.String())
		os.Exit(1)
	}

	rules := &rules.Rules{Functions: map[string]*rules.Rule{}}
	if err := json.Unmarshal(outb.Bytes(), rules); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	return rules
}

func main() {
	flags.ParseFlagsOrDie("Plugin config", &opts, nil)

	if err := os.Chdir(opts.Plugin); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	configFile, err := os.Open(".plzconfig")
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	defer configFile.Close()

	config := &core.Configuration{}
	gcfg.ReadInto(config, configFile)

	bs, err := json.Marshal(plugin.Plugin{
		Name:   config.PluginDefinition.Name,
		Help:   config.PluginDefinition.Description,
		Github: config.PluginDefinition.DocumentationSite,
		Config: getConfigFields(config),
		Rules:  getRules(config.PluginDefinition.BuildDefsDir),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	fmt.Println(string(bs))
}
