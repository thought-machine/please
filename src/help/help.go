// Package help prints help messages about parts of plz.
package help

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/parse/asp"
	"github.com/thought-machine/please/src/plz"
)

const topicsHelpMessage = `
The following help topics are available:

%s`

// maxSuggestionDistance is the maximum Levenshtein edit distance we'll suggest help topics at.
const maxSuggestionDistance = 4

// Help prints help on a particular topic.
// It returns true if the topic is known or false if it isn't.
func Help(topic string) bool {
	config, err := core.ReadDefaultConfigFiles(core.HostFS(), nil)
	if err != nil {
		// Don't bother the user if we can't load config files or whatever - just do our best.
		config = core.DefaultConfiguration()
	}

	if message := help(topic, config); message != "" {
		printMessage(message)
		return true
	}
	fmt.Printf("Sorry OP, can't halp you with %s\n", topic)
	if message := suggest(topic, config); message != "" {
		printMessage(message)
		fmt.Printf(" Or have a look on the website: https://please.build\n")
	} else {
		fmt.Printf("\nMaybe have a look on the website? https://please.build\n")
	}
	return false
}

// Topics prints the list of help topics beginning with the given prefix.
func Topics(prefix string, config *core.Configuration) {
	for _, topic := range allTopics(prefix, config) {
		fmt.Println(topic)
	}
}

func help(topic string, config *core.Configuration) string {
	topic = strings.ToLower(topic)
	if topic == "topics" {
		return fmt.Sprintf(topicsHelpMessage, strings.Join(allTopics("", config), "\n"))
	}
	for _, section := range []helpSection{allConfigHelp(config), miscTopics} {
		if message, found := section.Topics[topic]; found {
			message = strings.TrimSpace(message)
			if section.Preamble == "" {
				return message
			}
			return fmt.Sprintf(section.Preamble+"\n\n", topic) + message
		}
	}
	// Check built-in build rules.
	m := AllBuiltinFunctions(newState())
	if f, present := m[topic]; present {
		return helpFromBuildRule(f.FuncDef)
	}

	// Check plugins
	if pluginHelp := helpForPlugin(config, topic); pluginHelp != "" {
		return pluginHelp
	}

	return ""
}

// helpForPlugin returns some help text for a plugin
func helpForPlugin(config *core.Configuration, topic string) string {
	// Check if the topic is a plugin
	if _, ok := config.Plugin[topic]; ok {
		message := fmt.Sprintf("${BOLD_BLUE}%v${RESET} is a plugin defined in the ${GREEN}.plzconfig${RESET} file.\n", topic)

		buildLabel := config.Plugin[topic].Target
		if buildLabel.String() == "" {
			log.Fatalf("Plugin target must be specified in config")
		}
		subrepo := getSubrepoOrDie(topic, buildLabel)

		return pluginHelpMessage(subrepo, message)
	}

	// Check if the topic is a build rule defined in a plugin
	for pluginName, plugin := range config.Plugin {
		buildLabel := plugin.Target
		if buildLabel.String() == "" {
			log.Fatalf("Plugin target must be specified in config")
		}

		subrepo := getSubrepoOrDie(pluginName, buildLabel)
		buildRules := getPluginBuildDefs(subrepo)
		if rule, present := buildRules[topic]; present {
			return helpFromBuildRule(rule.FuncDef)
		}
	}

	return ""
}

// getSubrepoOrDie builds and returns a subrepo.
func getSubrepoOrDie(name string, target core.BuildLabel) *core.Subrepo {
	state := newState()

	// This is sufficient to get everything we need. The parsing of the build def files happens in getPluginBuildDefs.
	state.ParsePackageOnly = true

	plz.Run([]core.BuildLabel{target}, nil, state, state.Config, state.TargetArch)
	return state.Graph.SubrepoOrDie(name)
}

// formatConfigKey converts the config key from snake_case to CamelCase
func formatConfigKey(key string) string {
	key = strings.ToLower(key)
	parts := strings.Split(key, "_")
	for i, p := range parts {
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "")
}

// getPluginConfig returns a map of config options defined by a plugin
func getPluginConfig(subrepo *core.Subrepo) map[string]*core.PluginConfigDefinition {
	config := subrepo.State.RepoConfig
	return config.PluginConfig
}

func pluginHelpMessage(subrepo *core.Subrepo, message string) string {
	config := subrepo.State.Config
	description := config.PluginDefinition.Description
	docSite := config.PluginDefinition.DocumentationSite
	options := getPluginConfig(subrepo)
	buildRules := getPluginBuildDefs(subrepo)

	return formatPluginHelpMessage(message,
		description,
		docSite,
		options,
		buildRules,
	)
}

// formatPluginHelpMessage formats plugin information for the help command
func formatPluginHelpMessage(message, description, docSite string, options map[string]*core.PluginConfigDefinition, buildRulesMap map[string]*asp.Statement) string {
	if description != "" {
		message += "\n" + description + "\n"
	}
	if docSite != "" {
		message += "\n" + docSite + "\n"
	}
	configOptions := ""
	for k, v := range options {
		valueType := v.Type
		if v.Type == "" {
			valueType = "string"
		}
		key := v.ConfigKey
		if key == "" {
			key = formatConfigKey(k)
		}
		def := ""
		if len(v.DefaultValue) == 1 {
			def = v.DefaultValue[0]
		} else if len(v.DefaultValue) > 0 {
			def = strings.Join(v.DefaultValue, ", ")
		}
		if def != "" {
			def = "default: " + def
		}
		if v.Optional {
			if def != "" {
				def = ", " + def
			}
			def = "optional" + def
		}
		if def != "" {
			def = " (" + def + ")"
		}
		configOptions += fmt.Sprintf("${BLUE}   %s${RESET} ${GREEN}(%s)${RESET}${WHITE}%s${RESET} %s\n",
			key,
			valueType,
			def,
			v.Help)
	}
	if configOptions != "" {
		message += "\n${BOLD_YELLOW}This plugin has the following options:${RESET}\n" + configOptions
	}

	buildDefs := ""
	for k, v := range buildRulesMap {
		buildDefs += fmt.Sprintf("${BLUE}   %v${RESET}", strings.ToLower(k))
		arglist := "("
		for i, arg := range v.FuncDef.Arguments {
			if i != len(v.FuncDef.Arguments)-1 {
				arglist += arg.Name + ", "
			} else {
				arglist += arg.Name + ")"
			}
		}
		buildDefs += fmt.Sprintf("${GREEN}%v${RESET}\n", arglist)
	}
	if buildDefs != "" {
		message += "\n${BOLD_YELLOW}And provides the following build defs:${RESET}\n" + buildDefs
	}

	return message
}

func getPluginBuildDefs(subrepo *core.Subrepo) map[string]*asp.Statement {
	var dirs []string
	if len(subrepo.State.Config.PluginDefinition.BuildDefsDir) > 0 {
		dirs = append(dirs, subrepo.State.Config.PluginDefinition.BuildDefsDir...)
	} else {
		// By default, check the build_defs dir in the plugin
		dirs = append(dirs, "build_defs")
	}

	p := asp.NewParser(subrepo.State)
	ret := make(map[string]*asp.Statement)
	for _, dir := range dirs {
		dirEntries, err := fs.ReadDir(subrepo.FS, dir)
		if err != nil {
			log.Warningf("Failed to read %s: %s", dir, err)
		}
		for _, entry := range dirEntries {
			if entry.IsDir() {
				continue
			}

			path := filepath.Join(dir, entry.Name())
			bs, err := fs.ReadFile(subrepo.FS, path)
			if err != nil {
				log.Warningf("Failed to read %s: %s", path, err)
			}

			stmts, err := p.ParseData(bs, path)
			if err != nil {
				log.Warningf("Failed to parse %s: %s", path, err)
			}

			addAllFunctions(ret, stmts, false)
		}
	}

	return ret
}

// helpFromBuildRule returns the printable help message from a build rule (a function).
func helpFromBuildRule(f *asp.FuncDef) string {
	var b strings.Builder
	if err := template.Must(template.New("").Funcs(template.FuncMap{
		"trim": func(s string) string { return strings.Trim(s, `"`) },
	}).Parse(docstringTemplate)).Execute(&b, f); err != nil {
		log.Fatalf("%s", err)
	}
	s := strings.Replace(b.String(), "    Args:\n", "    ${BOLD_YELLOW}Args:${RESET}\n", 1)
	for _, a := range f.Arguments {
		r := regexp.MustCompile("( +)(" + a.Name + `)( \([a-z |]+\))?:`)
		s = r.ReplaceAllString(s, "$1$${YELLOW}$2$${RESET}$${GREEN}$3$${RESET}:")
	}
	return s
}

// suggest looks through all known help topics and tries to make a suggestion about what the user might have meant.
func suggest(topic string, config *core.Configuration) string {
	return cli.PrettyPrintSuggestion(topic, allTopics("", config), maxSuggestionDistance)
}

// allTopics returns all the possible topics to get help on.
func allTopics(prefix string, config *core.Configuration) []string {
	topics := []string{}

	// Config options
	for _, section := range []helpSection{allConfigHelp(config), miscTopics} {
		for t := range section.Topics {
			if strings.HasPrefix(t, prefix) {
				topics = append(topics, t)
			}
		}
	}

	// Built-in rules
	for t := range AllBuiltinFunctions(newState()) {
		if strings.HasPrefix(t, prefix) {
			topics = append(topics, t)
		}
	}

	// Plugins
	for pluginName, plugin := range config.Plugin {
		if strings.HasPrefix(pluginName, prefix) {
			topics = append(topics, pluginName)
		}
		subrepo := getSubrepoOrDie(pluginName, plugin.Target)
		for k := range getPluginConfig(subrepo) {
			if strings.HasPrefix(k, prefix) {
				topics = append(topics, k)
			}
		}
		for k := range getPluginBuildDefs(subrepo) {
			if strings.HasPrefix(k, prefix) {
				topics = append(topics, k)
			}
		}
	}

	sort.Strings(topics)
	return topics
}

type helpSection struct {
	Preamble string
	Topics   map[string]string
}

// printMessage prints a message, with some string replacements for ANSI codes.
func printMessage(msg string) {
	if cli.ShowColouredOutput {
		backtickRegex := regexp.MustCompile("\\`[^\\`\n]+\\`")
		msg = backtickRegex.ReplaceAllStringFunc(msg, func(s string) string {
			return "${BOLD_CYAN}" + strings.ReplaceAll(s, "`", "") + "${RESET}"
		})
	}
	// Replace % to %% when not followed by anything so it doesn't become a replacement.
	cli.Fprintf(os.Stdout, strings.ReplaceAll(msg, "% ", "%% ")+"\n")
}

const docstringTemplate = `${BLUE}{{ .Name }}${RESET} is
{{- if .IsBuiltin }} a built-in build rule in Please.
{{- else }} an add-on build rule for Please${RESET}.
{{- end }} Instructions for use & its arguments:

${BOLD_YELLOW}{{ .Name }}${RESET}(
{{- range $i, $a := .Arguments }}{{ if gt $i 0 }}, {{ end }}${GREEN}{{ $a.Name }}${RESET}{{ end -}}
):

{{ trim .Docstring }}
{{ if .IsBuiltin }}
Online help is available at https://please.build/lexicon.html#{{ .Name }}.
{{- end }}
`
