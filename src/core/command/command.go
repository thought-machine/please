package command

import (
	"fmt"
	"strings"

	"github.com/thought-machine/please/src/core"
)

type Command struct {
	tokens []token
	labels []core.BuildLabel
}

func (c *Command) String(state *core.BuildState, target *core.BuildTarget) string {
	// Validate that the target depends on the given inputs
	//TODO(jpoole): THis might not be the best approach. $(out_location ...) should require target has l as data
	// also what about tools?
	for _, l := range c.labels {
		if !target.HasDependency(l) {
			panic(fmt.Sprintf("Cannot expand command as %v does not depend on %v", target.Label, l))
		}
	}

	cmd := ""
	for _, tok := range c.tokens {
		cmd += tok.String(state)
	}
	return cmd
}

// Encloses the given string in quotes if needed.
func quote(s string) string {
	if strings.ContainsAny(s, "|&;()<>") {
		return "\"" + s + "\""
	}
	return s
}
