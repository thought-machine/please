package output

import (
	"fmt"
	"os"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
)

// initPrintf sets up the replacements used by printf.
func initPrintf(config *core.Configuration) {
	if !cli.ShowColouredOutput {
		replacements = map[string]string{}
	} else {
		if config.Display.ColourScheme == "light" {
			for k, v := range lightOverrides {
				replacements[k] = v
			}
		}
		for k, v := range config.Colours {
			replacements[k] = v
		}
	}
}

// printf is used throughout this package to print something to stderr with some
// replacements for pseudo-shell variables for ANSI formatting codes.
func printf(format string, args ...interface{}) {
	fmt.Fprint(os.Stderr, os.Expand(fmt.Sprintf(format, args...), replace))
}

func replace(s string) string {
	if repl, present := replacements[s]; present {
		return repl
	}
	return "$" + s
}

// These are the standard set of replacements we use.
var replacements = map[string]string{
	"BOLD":         "\x1b[1m",
	"BOLD_GREY":    "\x1b[30;1m",
	"BOLD_RED":     "\x1b[31;1m",
	"BOLD_GREEN":   "\x1b[32;1m",
	"BOLD_YELLOW":  "\x1b[33;1m",
	"BOLD_BLUE":    "\x1b[34;1m",
	"BOLD_MAGENTA": "\x1b[35;1m",
	"BOLD_CYAN":    "\x1b[36;1m",
	"BOLD_WHITE":   "\x1b[37;1m",
	"GREY":         "\x1b[30m",
	"RED":          "\x1b[31m",
	"GREEN":        "\x1b[32m",
	"YELLOW":       "\x1b[33m",
	"BLUE":         "\x1b[34m",
	"MAGENTA":      "\x1b[35m",
	"CYAN":         "\x1b[36m",
	"WHITE":        "\x1b[37m",
	"WHITE_ON_RED": "\x1b[37;41;1m",
	"RED_NO_BG":    "\x1b[31;49;1m",
	"RESET":        "\x1b[0m",
	"ERASE_AFTER":  "\x1b[K",
	"CLEAR_END":    "\x1b[0J",
}

// replacements overrides for light colour scheme.
var lightOverrides = map[string]string{
	"BOLD_GREY":  "\x1b[37;1m",
	"BOLD_WHITE": "\x1b[30;1m",
	"GREY":       "\x1b[37m",
	"WHITE":      "\x1b[30m",
}
