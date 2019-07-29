// +build !bootstrap

package core

import (
	"github.com/jessevdk/go-flags"

	"strings"
)

// AttachAliasFlags attaches the alias flags to the given flag parser.
// It returns true if any modifications were made.
func (config *Configuration) AttachAliasFlags(parser *flags.Parser) bool {
	for name, alias := range config.Alias {
		cmd := parser.Command
		fields := strings.Fields(name)
		for i, namePart := range fields {
			cmd = addSubcommand(cmd, namePart, alias.Desc, alias.PositionalLabels && len(alias.Subcommand) == 0 && i == len(fields)-1)
			for _, subcommand := range alias.Subcommand {
				addSubcommands(cmd, strings.Fields(subcommand), alias.PositionalLabels)
			}
			for _, flag := range alias.Flag {
				// This is unavailable during bootstrap due to being a local modification.
				cmd.AddOption(getOption(flag))
			}
		}
	}
	return len(config.Alias) > 0
}

// addSubcommands attaches a series of subcommands to the given command.
func addSubcommands(cmd *flags.Command, subcommands []string, positionalLabels bool) {
	if len(subcommands) > 0 && cmd != nil {
		addSubcommands(addSubcommand(cmd, subcommands[0], "", positionalLabels), subcommands[1:], positionalLabels)
	}
}

// addSubcommand adds a single subcommand to the given command.
// If one by that name already exists, it is returned.
func addSubcommand(cmd *flags.Command, subcommand, desc string, positionalLabels bool) *flags.Command {
	if existing := cmd.Find(subcommand); existing != nil {
		return existing
	}
	var data interface{} = &struct{}{}
	if positionalLabels {
		data = &struct {
			Args struct {
				Target []BuildLabel `positional-arg-name:"target" description:"Build targets"`
			} `positional-args:"true"`
		}{}
	}
	newCmd, _ := cmd.AddCommand(subcommand, desc, desc, data)
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
