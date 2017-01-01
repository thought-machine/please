// +build nobootstrap

// Package help prints help messages about parts of plz.
package help

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/op/go-logging.v1"

	"utils"
)

var log = logging.MustGetLogger("help")

// rulesPreamble is a message we print out before help for any built-in build rules.
const rulesPreamble = `
%s is a built-in build rule in Please. Instructions for use & its arguments:

`
const configPreamble = `
%s is a config setting defined in the .plzconfig file. See "plz help plzconfig" for more information.

`
const miscPreamble = `
%s is a general Please concept.

`

var allHelpFiles = []string{"rule_defs.json", "config.json", "misc.json"}
var allHelpPreambles = []string{rulesPreamble, configPreamble, miscPreamble}

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

func help(topic string) string {
	topic = strings.ToLower(topic)
	for i, filename := range allHelpFiles {
		if message, found := halp(topic, filename, allHelpPreambles[i]); found {
			return message
		}
	}
	return ""
}

func halp(topic, filename, preamble string) (string, bool) {
	m := loadData(filename)
	message, found := m[topic]
	if !found {
		return "", false
	}
	return fmt.Sprintf(preamble, topic) + message, true
}

func loadData(filename string) map[string]string {
	log.Debug("Opening help file %s", filename)
	data := MustAsset(filename)
	m := map[string]string{}
	if err := json.Unmarshal(data, &m); err != nil {
		log.Fatalf("Failed to load help data: %s\n", err)
	}
	return m
}

// suggest looks through all known help topics and tries to make a suggestion about what the user might have meant.
func suggest(topic string) string {
	topics := []string{}
	for _, filename := range allHelpFiles {
		for t := range loadData(filename) {
			topics = append(topics, t)
		}
	}
	return utils.SuggestMessage(topic, topics, maxSuggestionDistance)
}
