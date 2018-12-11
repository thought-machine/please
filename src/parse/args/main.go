// Package main implements a simple binary to print out builtin rules.
// This all seems pretty fiddly and complex at this point; it avoids circular
// dependencies but also seems unnecessary given that ultimately plz has a command
// to do this itself.
package main

import (
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/parse"
)

func main() {
	parse.PrintRuleArgs(core.NewDefaultBuildState(), nil)
}
