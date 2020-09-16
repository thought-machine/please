package command

import (
	"path"
	"strings"

	"github.com/thought-machine/please/src/core"
)

type Command struct {
	tokens []token
}

func (c *Command) String(state *core.BuildState, target *core.BuildTarget) string {
	cmd := ""
	for _, tok := range c.tokens {
		cmd += tok.String(state, target)
	}
	return cmd
}


func fileDestination(target, dep *core.BuildTarget, out string, dir, outPrefix, test bool) string {
	if outPrefix {
		return handleDir(dep.OutDir(), out, dir)
	}
	if test && target == dep {
		// Slightly fiddly case because tests put binaries in a possibly slightly unusual place.
		return "./" + out
	}
	return handleDir(dep.Label.PackageName, out, dir)
}

// Encloses the given string in quotes if needed.
func quote(s string) string {
	if strings.ContainsAny(s, "|&;()<>") {
		return "\"" + s + "\""
	}
	return s
}

// handleDir chooses either the out dir or the actual output location depending on the 'dir' flag.
func handleDir(outDir, output string, dir bool) string {
	if dir {
		return outDir
	}
	return path.Join(outDir, output)
}
