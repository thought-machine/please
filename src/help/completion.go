package help

import (
	"github.com/thought-machine/go-flags"
	"github.com/thought-machine/please/src/core"
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
	topics := allTopics(match, core.DefaultConfiguration())
	completions := make([]flags.Completion, len(topics))
	for i, topic := range topics {
		completions[i].Item = topic
	}
	return completions
}
