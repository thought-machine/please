// +build !bootstrap

package core

import (
	"github.com/jessevdk/go-flags"

	"strings"
)

// AttachAliasFlags attaches the alias flags to the given flag parser.
// It returns true if any modifications were made.
func (config *Configuration) AttachAliasFlags(parser *flags.Parser) bool {
	for name, alias := range config.AllAliases() {
		cmd := parser.Command
		for _, namePart := range strings.Fields(name) {
			cmd = addSubcommand(cmd, namePart, alias.Desc)
			for _, subcommand := range alias.Subcommand {
				addSubcommands(cmd, strings.Fields(subcommand))
			}
			for _, flag := range alias.Flag {
				// This is unavailable during bootstrap due to being a local modification.
				cmd.AddOption(getOption(flag))
			}
		}
	}
	return len(config.Aliases) > 0 || len(config.Alias) > 0
}

// addSubcommands attaches a series of subcommands to the given command.
func addSubcommands(cmd *flags.Command, subcommands []string) {
	if len(subcommands) > 0 && cmd != nil {
		addSubcommands(addSubcommand(cmd, subcommands[0], ""), subcommands[1:])
	}
}

// addSubcommand adds a single subcommand to the given command.
// If one by that name already exists, it is returned.
func addSubcommand(cmd *flags.Command, subcommand, desc string) *flags.Command {
	if existing := cmd.Find(subcommand); existing != nil {
		return existing
	}
	newCmd, _ := cmd.AddCommand(subcommand, desc, desc, &struct{}{})
	return newCmd
}

// getOption creates a new flags.Option.
// This is a fiddle since it doesn't really expose a direct way of doing this programmatically.
func getOption(name string) *flags.Option {
	data := struct {
		Opt string `long:"option"`
	}{}
	p := flags.NewParser(&data, 0)
	opt := p.FindOptionByLongName("option")
	opt.LongName = strings.TrimLeft(name, "-")
	if len(name) == 2 && name[0] == '-' {
		opt.ShortName = rune(name[1])
	}
	return opt
}
