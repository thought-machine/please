// +build nobootstrap

// Package help prints help messages about parts of plz.
package help

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/op/go-logging.v1"

	"utils"
)

var log = logging.MustGetLogger("help")

const defaultHelpMessage = `
Please is a high-performance language-agnostic build system.

Try plz help <topic> for help on a specific topic;
plz --help if you want information on flags / options / commands that it accepts;
plz help topics if you want to see the list of possible topics to get help on
or try a few commands like plz build or plz test if your repo is already set up
and you'd like to see it in action.

Or see the website (https://please.build) for more information.
`
const topicsHelpMessage = `
The following help topics are available:

%s`

var allHelpFiles = []string{"rule_defs.json", "config.json", "misc.json"}

// maxSuggestionDistance is the maximum Levenshtein edit distance we'll suggest help topics at.
const maxSuggestionDistance = 5

// Help prints help on a particular topic.
// It returns true if the topic is known or false if it isn't.
func Help(topic string) bool {
	if message := help(topic); message != "" {
		fmt.Println(message)
		return true
	}
	fmt.Printf("Sorry OP, can't halp you with %s\n", topic)
	if message := suggest(topic); message != "" {
		fmt.Println(message)
		fmt.Printf("Or have a look on the website: https://please.build\n")
	} else {
		fmt.Printf("\nMaybe have a look on the website? https://please.build\n")
	}
	return false
}

// HelpTopics prints the list of help topics beginning with the given prefix.
func HelpTopics(prefix string) {
	for _, topic := range allTopics() {
		if strings.HasPrefix(topic, prefix) {
			fmt.Println(topic)
		}
	}
}

func help(topic string) string {
	topic = strings.ToLower(topic)
	if topic == "" {
		return defaultHelpMessage
	} else if topic == "topics" {
		return fmt.Sprintf(topicsHelpMessage, strings.Join(allTopics(), "\n"))
	}
	for _, filename := range allHelpFiles {
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
	return fmt.Sprintf(preamble+"\n\n", topic) + message, true
}

func loadData(filename string) (string, map[string]string) {
	log.Debug("Opening help file %s", filename)
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
	for _, filename := range allHelpFiles {
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
