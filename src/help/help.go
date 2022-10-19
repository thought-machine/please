// Package help prints help messages about parts of plz.
package help

import (
	"fmt"
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
	if message := help(topic); message != "" {
		printMessage(message)
		return true
	}
	fmt.Printf("Sorry OP, can't halp you with %s\n", topic)
	if message := suggest(topic); message != "" {
		printMessage(message)
		fmt.Printf(" Or have a look on the website: https://please.build\n")
	} else {
		fmt.Printf("\nMaybe have a look on the website? https://please.build\n")
	}
	return false
}

// Topics prints the list of help topics beginning with the given prefix.
func Topics(prefix string) {
	for _, topic := range allTopics(prefix) {
		fmt.Println(topic)
	}
}

func help(topic string) string {
	config, err := core.ReadDefaultConfigFiles(nil)
	if err != nil {
		// Don't bother the user if we can't load config files or whatever - just do our best.
		config = core.DefaultConfiguration()
	}

	topic = strings.ToLower(topic)
	if topic == "topics" {
		return fmt.Sprintf(topicsHelpMessage, strings.Join(allTopics(""), "\n"))
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
	if _, ok := config.Plugin[topic]; ok {
		message := fmt.Sprintf("${BOLD_BLUE}%v${RESET} is a plugin defined in the ${GREEN}.plzconfig${RESET} file.\n", topic)

		buildLabel := config.Plugin[topic].Target
		if buildLabel.String() == "" {
			log.Fatalf("Plugin target must be specified in config")
		}
		state := newState()

		// Parse the subrepo (Run reads the plugin config into config)
		plz.Run([]core.BuildLabel{buildLabel}, nil, state, config, state.TargetArch)
		subrepo := state.Graph.Subrepo(topic)
		if subrepo == nil {
			log.Fatalf("Tried to get subrepo %v but failed", topic)
		}

		return getPluginOptionsAndBuildDefs(subrepo, message)
	}
	return ""
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

// getPluginOptionsAndBuildDefs looks for information for the plugin specified in the config file
func getPluginOptionsAndBuildDefs(subrepo *core.Subrepo, message string) string {
	config := subrepo.State.Config
	if config.PluginDefinition.Description != "" {
		message += "\n" + config.PluginDefinition.Description + "\n"
	}
	if config.PluginDefinition.DocumentationSite != "" {
		message += "\n" + config.PluginDefinition.DocumentationSite + "\n"
	}
	configOptions := ""
	// Ensure these come out in the same order
	keys := make([]string, 0, len(config.PluginConfig))
	for k := range config.PluginConfig {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := config.PluginConfig[k]
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

	buildFuncMap := populatePluginBuildFuncs(subrepo)
	buildDefs := ""
	for k, v := range buildFuncMap {
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

func populatePluginBuildFuncs(subrepo *core.Subrepo) map[string]*asp.Statement {
	p := asp.NewParser(subrepo.State)
	var dirs []string
	if len(subrepo.State.Config.PluginDefinition.BuildDefsDir) > 0 {
		for _, dir := range subrepo.State.Config.PluginDefinition.BuildDefsDir {
			dirs = append(dirs, filepath.Join(subrepo.Root, dir))
		}
	} else {
		// By default, check the build_defs dir in the plugin
		dirs = append(dirs, filepath.Join(subrepo.Root, "build_defs"))
	}

	ret := make(map[string]*asp.Statement)
	for _, dir := range dirs {
		if files, err := os.ReadDir(dir); err == nil {
			for _, file := range files {
				if !file.IsDir() {
					if stmts, err := p.ParseFileOnly(filepath.Join(dir, file.Name())); err == nil {
						addAllFunctions(ret, stmts, false)
					}
				}
			}
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
func suggest(topic string) string {
	return cli.PrettyPrintSuggestion(topic, allTopics(""), maxSuggestionDistance)
}

// allTopics returns all the possible topics to get help on.
func allTopics(prefix string) []string {
	topics := []string{}
	for _, section := range []helpSection{allConfigHelp(core.DefaultConfiguration()), miscTopics} {
		for t := range section.Topics {
			if strings.HasPrefix(t, prefix) {
				topics = append(topics, t)
			}
		}
	}
	for t := range AllBuiltinFunctions(newState()) {
		if strings.HasPrefix(t, prefix) {
			topics = append(topics, t)
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
{{- else }} an add-on build rule for Please defined in ${YELLOW}{{ .EoDef.Filename }}${RESET}.
{{- end }} Instructions for use & its arguments:

${BOLD_YELLOW}{{ .Name }}${RESET}(
{{- range $i, $a := .Arguments }}{{ if gt $i 0 }}, {{ end }}${GREEN}{{ $a.Name }}${RESET}{{ end -}}
):

{{ trim .Docstring }}
{{ if .IsBuiltin }}
Online help is available at https://please.build/lexicon.html#{{ .Name }}.
{{- end }}
`
