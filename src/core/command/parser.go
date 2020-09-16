package command

import (
	"fmt"
	"github.com/thought-machine/please/src/core"
	"regexp"
	"strings"
)

var keywords = []string{
	"location", "locations", "out_location", "exe", "out_exe", "dir", "dir", "hash", "worker",
}

// Matches keywords within a command: $(<keyword> ...)
var keywordRegex = regexp.MustCompile(fmt.Sprintf("\\$\\((%v) [^\\)]+\\)", strings.Join(keywords, "|")))

func parse(cmd string, packageName string) Command {
	command := Command{
		tokens: []token{},
	}

	for {
		loc := keywordRegex.FindIndex([]byte(cmd))
		if loc == nil {
			break
		}

		start := loc[0]
		end := loc[1]

		command.tokens = append(command.tokens, bash(cmd[:start]))

		match := cmd[start:end]
		cmd = cmd[end:]

		args := strings.Split(strings.TrimSuffix(strings.TrimPrefix(match, "$("), ")"), " ")

		keyword, args := args[0], args[1:]
		command.tokens = append(command.tokens, keywordToToken(keyword, args, packageName))
	}
	command.tokens = append(command.tokens, bash(cmd))

	return command
}

func keywordToToken(keyword string, args []string, packageName string) token {
	if keyword == "location" {
		//TODO(jpoole): move construction out of here and add validation
		return location(core.ParseBuildLabel(args[0], packageName))
	}
	if keyword == "locations" {
		return locations(core.ParseBuildLabel(args[0], packageName))
	}
	panic("TODO: implement keyword " + keyword)
}