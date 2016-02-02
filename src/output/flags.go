// Contains helper functions related to flag parsing.

package output

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/jessevdk/go-flags"

	"core"
)

// ParseFlags parses the app's flags and returns the parser, any extra arguments, and any error encountered.
// It may exit if certain options are encountered (eg. --help).
func ParseFlags(appname string, data interface{}, args []string) (*flags.Parser, []string, error) {
	parser := flags.NewNamedParser(path.Base(args[0]), flags.HelpFlag|flags.PassDoubleDash)
	parser.AddGroup(appname+" options", "", data)
	extraArgs, err := parser.ParseArgs(args[1:])
	if err != nil {
		if err.(*flags.Error).Type == flags.ErrHelp {
			fmt.Printf("%s\n", err)
			os.Exit(0)
		} else if err.(*flags.Error).Type == flags.ErrUnknownFlag && strings.Contains(err.(*flags.Error).Message, "halp") {
			fmt.Printf("Hmmmmm, hows can I halp you?\n")
			parser.WriteHelp(os.Stderr)
			os.Exit(1)
		} else if v := reflect.ValueOf(data).Elem().FieldByName("Version"); v != reflect.ValueOf(nil) && v.Bool() {
			fmt.Printf("%s version %s\n", appname, core.PleaseVersion)
			os.Exit(0) // Ignore other errors if --version was passed.
		}
	}
	return parser, extraArgs, err
}

// ParseFlagsOrDie, as the name suggests, parses the app's flags and dies if unsuccessful.
// Also dies if any unexpected arguments are passed.
func ParseFlagsOrDie(appname string, data interface{}) *flags.Parser {
	return ParseFlagsFromArgsOrDie(appname, data, os.Args)
}

// ParseFlagsFromArgsOrDie is similar to ParseFlagsOrDie but allows control over the
// flags passed.
func ParseFlagsFromArgsOrDie(appname string, data interface{}, args []string) *flags.Parser {
	parser, extraArgs, err := ParseFlags(appname, data, args)
	if err != nil {
		parser.WriteHelp(os.Stderr)
		fmt.Printf("\n%s\n", err)
		os.Exit(1);
	} else if len(extraArgs) > 0 {
		fmt.Printf("Unknown option %s\n", extraArgs)
		parser.WriteHelp(os.Stderr)
		os.Exit(1)
	}
	return parser
}

// A ByteSize is used for flags that represent some quantity of bytes that can be
// passed as human-readable quantities (eg. "10G").
type ByteSize int64

func (b *ByteSize) UnmarshalFlag(in string) error {
	b2, err := humanize.ParseBytes(in)
	*b = ByteSize(b2)
	return err
}
