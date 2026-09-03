package query

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/please-build/gcfg"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/plz"
)

// Config prints configuration settings in human-readable format.
func Config(config *core.Configuration, options []string) {
	if len(options) == 0 {
		v, err := gcfg.Stringify(config)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Print(v)
	} else {
		for _, option := range options {
			section, subsection, name, err := parseOption(option)
			if err != nil {
				log.Fatal(err)
			}

			values, err := gcfg.Get(config, section, subsection, name)
			if err != nil {
				if section == "plugin" {
					if defaultValues, ok := pluginConfigFieldDefaultValues(config, subsection, name); ok {
						values = defaultValues
						err = nil
					}
				}
				if err != nil {
					log.Fatalf("Failed to get %s: %s", option, err)
				}
			}

			for _, value := range values {
				fmt.Println(value)
			}
		}
	}
}

// pluginConfigFieldDefaultValues returns the default values of a plugin config field and a boolean indicating whether
// the plugin field actually exists.
func pluginConfigFieldDefaultValues(config *core.Configuration, pluginName string, name string) ([]string, bool) {
	plugin, ok := config.Plugin[pluginName]
	if !ok {
		return nil, false
	}

	state := core.NewBuildState(config)
	plz.Run([]core.BuildLabel{plugin.Target}, nil, state, state.Config, state.TargetArch)
	subrepo := state.Graph.SubrepoOrDie(pluginName)
	subrepo.State.Initialise(subrepo)

	for key, field := range subrepo.State.RepoConfig.PluginConfig {
		configKey := field.ConfigKey
		if configKey == "" {
			configKey = strings.ReplaceAll(key, "_", "")
		}
		if strings.EqualFold(configKey, name) {
			values := normaliseBuildLabels(field.DefaultValue, subrepo.Name)
			return values, true
		}
	}

	return nil, false
}

// normaliseBuildLabels returns a copy of values with each build label made absolute.
// For example, //tools/bar in subrepo foo is replaced by ///foo//tools/bar.
func normaliseBuildLabels(values []string, subrepo string) []string {
	valuesCopy := make([]string, len(values))
	for i, value := range values {
		if core.LooksLikeABuildLabel(value) {
			target, annotation := core.SplitLabelAnnotation(value)
			if label, err := core.TryParseBuildLabel(target, "", subrepo); err == nil {
				annotatedLabel := core.AnnotatedOutputLabel{
					BuildLabel: label,
					Annotation: annotation,
				}
				value = annotatedLabel.String()
			}
		}
		valuesCopy[i] = value
	}
	return valuesCopy
}

// ConfigJSON prints the configuration settings as JSON.
func ConfigJSON(config *core.Configuration) {
	data, err := gcfg.RawJSON(config)
	if err != nil {
		log.Fatalf("Failed to get JSON configuration: %s", err)
	}

	var out bytes.Buffer
	if err := json.Indent(&out, data, "", "    "); err != nil {
		log.Fatalf("Failed to parse JSON configuration: %s", err)
	}

	fmt.Print(out.String())
}

func parseOption(option string) (section, subsection, name string, err error) {
	parts := strings.Split(option, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return "", "", "", fmt.Errorf("Bad option format. Example: section.subsection.name or section.name")
	}
	if len(parts) == 2 {
		return parts[0], "", parts[1], nil
	}
	return parts[0], parts[1], parts[2], nil
}
