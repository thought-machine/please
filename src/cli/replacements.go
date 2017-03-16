package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
)

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
	"RESET":        "\x1b[0m",
	"RESETLN":      "\x1b[1G\x1b[2K", // Resets back to start of line and clears it.
}

// Printf is a convenience wrapper to Fprintf that always writes to stderr.
func Printf(msg string, args ...interface{}) {
	Fprintf(os.Stderr, msg, args...)
}

// Fprintf implements essentially fmt.Fprintf with replacements of
// some ANSI sequences, e.g. ${BOLD_RED} -> \x1bwhatever.
func Fprintf(w io.Writer, msg string, args ...interface{}) {
	for k, v := range replacements {
		if !StdErrIsATerminal || !StdOutIsATerminal {
			v = ""
		}
		msg = strings.Replace(msg, fmt.Sprintf("${%s}", k), v, -1)
	}
	fmt.Fprintf(w, msg, args...)
}
