// +build nobootstrap

// Package help prints help messages about parts of plz.
package help

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/op/go-logging.v1"

	"cli"
	"utils"
)

var log = logging.MustGetLogger("help")

const topicsHelpMessage = `
The following help topics are available:

%s`

// maxSuggestionDistance is the maximum Levenshtein edit distance we'll suggest help topics at.
const maxSuggestionDistance = 5

var backtickRegex = regexp.MustCompile("\\`[^\\`\n]+\\`")

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
	for _, topic := range allTopics() {
		if strings.HasPrefix(topic, prefix) {
			fmt.Println(topic)
		}
	}
}

func help(topic string) string {
	topic = strings.ToLower(topic)
	if topic == "topics" {
		return fmt.Sprintf(topicsHelpMessage, strings.Join(allTopics(), "\n"))
	}
	for _, filename := range AssetNames() {
		if message, found := findHelpFromFile(topic, filename); found {
			return message
		}
	}
	return ""
}

func findHelpFromFile(topic, filename string) (string, bool) {
	preamble, topics := loadData(filename)
	message, found := topics[topic]
	if !found {
		return "", false
	}
	if preamble == "" {
		return message, true
	}
	return fmt.Sprintf(preamble+"\n\n", topic) + message, true
}

func loadData(filename string) (string, map[string]string) {
	data := MustAsset(filename)
	f := helpFile{}
	if err := json.Unmarshal(data, &f); err != nil {
		log.Fatalf("Failed to load help data: %s\n", err)
	}
	return f.Preamble, f.Topics
}

// suggest looks through all known help topics and tries to make a suggestion about what the user might have meant.
func suggest(topic string) string {
	return utils.PrettyPrintSuggestion(topic, allTopics(), maxSuggestionDistance)
}

// allTopics returns all the possible topics to get help on.
func allTopics() []string {
	topics := []string{}
	for _, filename := range AssetNames() {
		_, data := loadData(filename)
		for t := range data {
			topics = append(topics, t)
		}
	}
	sort.Strings(topics)
	return topics
}

// helpFile is a struct we use for unmarshalling.
type helpFile struct {
	Preamble string            `json:"preamble"`
	Topics   map[string]string `json:"topics"`
}

// printMessage prints a message, with some string replacements for ANSI codes.
func printMessage(msg string) {
	if cli.StdErrIsATerminal && cli.StdOutIsATerminal {
		msg = backtickRegex.ReplaceAllStringFunc(msg, func(s string) string {
			return "${BOLD_CYAN}" + strings.Replace(s, "`", "", -1) + "${RESET}"
		})
	}
	// Replace % to %% when not followed by anything so it doesn't become a replacement.
	cli.Fprintf(os.Stdout, strings.Replace(msg, "% ", "%% ", -1))
}
