// +build nobootstrap

package help

import (
	"strings"

	"github.com/jessevdk/go-flags"
)

// Topic is a help topic that implements completion for flags.
// It's otherwise equivalent to a string.
type Topic string

// UnmarshalFlag implements the flags.Unmarshaler interface
func (topic *Topic) UnmarshalFlag(value string) error {
	*topic = Topic(value)
	return nil
}

// Complete implements the flags.Completer interface, which is used for shell completion.
func (topic Topic) Complete(match string) []flags.Completion {
	topics := allTopics()
	completions := make([]flags.Completion, len(topics))
	for i, topic := range topics {
		if strings.HasPrefix(topic, match) {
			completions[i].Item = topic
		}
	}
	return completions
}
