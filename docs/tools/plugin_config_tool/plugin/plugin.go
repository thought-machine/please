package plugin

import "github.com/thought-machine/please/docs/tools/lexicon_templater/rules"

type Plugin struct {
	Name, ID, Help, Codelab, Github string

	Config []*ConfigField
	Rules  *rules.Rules
}

type ConfigField struct {
	Name, Type, Help, DefaultValue string
	Inherit, Repeatable, Defaults, Optional bool
}
