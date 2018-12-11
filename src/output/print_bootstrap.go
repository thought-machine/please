// +build bootstrap

package output

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/thought-machine/please/src/cli"
)

// Used to strip the formatting stuff at bootstrap time.
var stripFormatting = regexp.MustCompile(`\$\{[^\}]+\}`)

// printf is the bootstrap version with some niceties to handle the custom escape sequences.
func printf(format string, args ...interface{}) {
	msg := strings.Replace(fmt.Sprintf(format, args...), "${ERASE_AFTER}", "\x1b[K", -1)
	msg = stripFormatting.ReplaceAllString(msg, "")
	fmt.Fprint(os.Stderr, cli.StripAnsi.ReplaceAllString(msg, ""))
}
