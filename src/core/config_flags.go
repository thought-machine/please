package core

import (
	"github.com/thought-machine/go-flags"

	"strings"
)

// AttachAliasFlags attaches the alias flags to the given flag parser.
// It returns true if any modifications were made.
func (config *Configuration) AttachAliasFlags(parser *flags.Parser, args []string) bool {
	for name, alias := range config.Alias {
		var err error
		cmd := parser.Command
		if alias.Config == "" {
			fields := strings.Fields(name)
			for i, namePart := range fields {
				cmd = addSubcommand(cmd, namePart, alias.Desc, alias.PositionalLabels && len(alias.Subcommand) == 0 && i == len(fields)-1)
				for _, subcommand := range alias.Subcommand {
					addSubcommands(cmd, strings.Fields(subcommand), alias.PositionalLabels)
				}
				for _, flag := range alias.Flag {
					var f struct {
						Data bool
					}
					cmd.AddOption(&flags.Option{
						LongName: strings.TrimLeft(flag, "-"),
					}, &f.Data)
				}
			}
		} else {
			// throw exception if any fields other than desc and config exist in this alias
			if alias.Cmd != "" || len(alias.Subcommand) != 0 || len(alias.Flag) != 0 {
				log.Warningf("config flags attach: %v, %v, %v, %v", name, alias.Cmd, len(alias.Subcommand), len(alias.Flag))
				log.Errorf("Alias config field duplication for alias %s: use one method of specification", name)
			}
			cmd, err = alias.ParseAliasConfigs(name, args, alias.Config, alias.Desc, cmd)
			if err != nil {
				log.Debugf("Error parsing alias config %v", err)
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
	data := useInterface(positionalLabels)
	newCmd, _ := cmd.AddCommand(subcommand, desc, desc, data)
	return newCmd
}

func useInterface(positionalLabels bool) interface{} {
	var data interface{} = &struct{}{}
	if positionalLabels {
		data = &struct {
			Args struct {
				Target []BuildLabel `positional-arg-name:"target" description:"Build targets"`
			} `positional-args:"true"`
		}{}
	}
	return data
}
