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
	config, err := core.ReadDefaultConfigFiles(core.HostFS(), nil)
	if err != nil {
		config = core.DefaultConfiguration()
	}

	topics := allTopics(match, config)
	completions := make([]flags.Completion, len(topics))
	for i, topic := range topics {
		completions[i].Item = topic
	}
	return completions
}
