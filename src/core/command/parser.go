package command

import (
	"fmt"
	"github.com/thought-machine/please/src/core"
	"regexp"
	"strings"
)

var keywords = []string{
	"location", "locations", "out_location", "exe", "out_exe", "dir", "hash",
}

// Matches keywords within a command: $(<keyword> ...)
var keywordRegex = regexp.MustCompile(fmt.Sprintf("\\$\\((%v) [^\\)]+\\)", strings.Join(keywords, "|")))

func parse(cmdStr string, packageName string) Command {
	cmd := Command{
		tokens: []token{},
		labels: []core.BuildLabel{},
	}

	for {
		loc := keywordRegex.FindIndex([]byte(cmdStr))
		if loc == nil {
			break
		}

		start := loc[0]
		end := loc[1]

		cmd.tokens = append(cmd.tokens, bash(cmdStr[:start]))

		match := cmdStr[start:end]
		cmdStr = cmdStr[end:]

		args := strings.Split(strings.TrimSuffix(strings.TrimPrefix(match, "$("), ")"), " ")

		keyword, args := args[0], args[1:]
		addKeywordToCommand(&cmd, keyword, args, packageName)
	}
	cmd.tokens = append(cmd.tokens, bash(cmdStr))

	return cmd
}

func addKeywordToCommand(cmd *Command, keyword string, args []string, packageName string) {
	l := core.ParseBuildLabel(args[0], packageName)

	//TODO(jpoole): move construction out of here and add validation
	if keyword == "location" {
		cmd.tokens = append(cmd.tokens, location(l))
	}
	if keyword == "locations" {
		cmd.tokens = append(cmd.tokens, locations(l))
	}
	if keyword == "out_location" {
		cmd.tokens = append(cmd.tokens, outLocation(l))
	}
	if keyword == "exe" {
		cmd.tokens = append(cmd.tokens, exe(l))
	}
	if keyword == "out_exe" {
		cmd.tokens = append(cmd.tokens, outExe(l))
	}
	//TODO(jpoole): this one isn't documented
	if keyword == "dir" {
		cmd.tokens = append(cmd.tokens, dir(l))
	}
	if keyword == "hash" {
		cmd.tokens = append(cmd.tokens, hash(l))
	}

	cmd.labels = append(cmd.labels, l)
}