// +build nobootstrap

// Package help prints help messages about parts of plz.
package help

import (
	"encoding/json"
	"fmt"

	"gopkg.in/op/go-logging.v1"
)

var log = logging.MustGetLogger("help")

// rulesPreamble is a message we print out before help for any built-in build rules.
const rulesPreamble = `
%s is a built-in build rule in Please. Instructions for use & its arguments:

`

// Help prints help on a particular topic.
// It returns true if the topic is known or false if it isn't.
func Help(topic string) bool {
	if message := help(topic); message != "" {
		fmt.Println(message)
		return true
	}
	fmt.Printf("Sorry OP, can't halp you with %s\n", topic)
	// TODO(pebers): do some fuzzy matching here
	fmt.Printf("\nMaybe have a look on the website: https://please.build\n")
	return false
}

func help(topic string) string {
	if message, found := halp(topic, "rule_defs.json", rulesPreamble); found {
		return message
	}
	return ""
}

func halp(topic, filename, preamble string) (string, bool) {
	data := MustAsset(filename)
	m := map[string]string{}
	if err := json.Unmarshal(data, &m); err != nil {
		log.Fatalf("Failed to load help data: %s\n", err)
	}
	message, found := m[topic]
	if !found {
		return "", false
	}
	return fmt.Sprintf(preamble, topic) + message, true
}
