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
		cmd, _ := parser.AddCommand(name, alias.Desc, alias.Desc, &struct{}{})
		for _, subcommand := range alias.Subcommand {
			addSubcommands(cmd, strings.Fields(subcommand))
		}
		for _, flag := range alias.Flag {
			// This is unavailable during bootstrap due to being a local modification.
			cmd.AddOption(getOption(flag))
		}
	}
	return len(config.Aliases) > 0 || len(config.Alias) > 0
}

// addSubcommands attaches a series of subcommands to the given command.
func addSubcommands(cmd *flags.Command, subcommand []string) {
	if len(subcommand) > 0 && cmd != nil {
		cmd2 := cmd.Find(subcommand[0])
		if cmd2 == nil {
			cmd2, _ = cmd.AddCommand(subcommand[0], "", "", &struct{}{})
		}
		addSubcommands(cmd2, subcommand[1:])
	}
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
