// Package help prints help messages about parts of plz.
package help

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/parse/asp"
	"github.com/thought-machine/please/src/utils"
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
	topic = strings.ToLower(topic)
	if topic == "topics" {
		return fmt.Sprintf(topicsHelpMessage, strings.Join(allTopics(""), "\n"))
	}
	for _, section := range []helpSection{allConfigHelp(), miscTopics} {
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
	return ""
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
	return utils.PrettyPrintSuggestion(topic, allTopics(""), maxSuggestionDistance)
}

// allTopics returns all the possible topics to get help on.
func allTopics(prefix string) []string {
	topics := []string{}
	for _, section := range []helpSection{allConfigHelp(), miscTopics} {
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
