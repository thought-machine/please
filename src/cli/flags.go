// Package cli contains helper functions related to flag parsing and logging.
package cli

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/go-semver/semver"
	"github.com/dustin/go-humanize"
	"github.com/jessevdk/go-flags"
)

// GiByte is a re-export for convenience of other things using it.
const GiByte = humanize.GiByte

// ParseFlags parses the app's flags and returns the parser, any extra arguments, and any error encountered.
// It may exit if certain options are encountered (eg. --help).
func ParseFlags(appname string, data interface{}, args []string) (*flags.Parser, []string, error) {
	parser := flags.NewNamedParser(path.Base(args[0]), flags.HelpFlag|flags.PassDoubleDash)
	parser.AddGroup(appname+" options", "", data)
	extraArgs, err := parser.ParseArgs(args[1:])
	if err != nil {
		if err.(*flags.Error).Type == flags.ErrHelp {
			writeUsage(data)
			fmt.Printf("%s\n", err)
			os.Exit(0)
		} else if err.(*flags.Error).Type == flags.ErrUnknownFlag && strings.Contains(err.(*flags.Error).Message, "`halp'") {
			fmt.Printf("Hmmmmm, hows can I halp you?\n")
			writeUsage(data)
			parser.WriteHelp(os.Stderr)
			os.Exit(0)
		}
	}
	return parser, extraArgs, err
}

// ParseFlagsOrDie parses the app's flags and dies if unsuccessful.
// Also dies if any unexpected arguments are passed.
func ParseFlagsOrDie(appname, version string, data interface{}) *flags.Parser {
	return ParseFlagsFromArgsOrDie(appname, version, data, os.Args)
}

// ParseFlagsFromArgsOrDie is similar to ParseFlagsOrDie but allows control over the
// flags passed.
func ParseFlagsFromArgsOrDie(appname, version string, data interface{}, args []string) *flags.Parser {
	parser, extraArgs, err := ParseFlags(appname, data, args)
	if err != nil && err.(*flags.Error).Type == flags.ErrUnknownFlag && strings.Contains(err.(*flags.Error).Message, "`version'") {
		fmt.Printf("%s version %s\n", appname, version)
		os.Exit(0) // Ignore other errors if --version was passed.
	}
	if err != nil {
		writeUsage(data)
		parser.WriteHelp(os.Stderr)
		fmt.Printf("\n%s\n", err)
		os.Exit(1)
	} else if len(extraArgs) > 0 {
		writeUsage(data)
		fmt.Printf("Unknown option %s\n", extraArgs)
		parser.WriteHelp(os.Stderr)
		os.Exit(1)
	}
	return parser
}

// writeUsage prints any usage specified on the flag struct.
func writeUsage(opts interface{}) {
	if s := getUsage(opts); s != "" {
		fmt.Println(s)
		fmt.Println("") // extra blank line
	}
}

// getUsage extracts any usage specified on a flag struct.
// It is set on a field named Usage, either by value or in a struct tag named usage.
func getUsage(opts interface{}) string {
	if field := reflect.ValueOf(opts).Elem().FieldByName("Usage"); field.IsValid() && field.String() != "" {
		return strings.TrimSpace(field.String())
	}
	if field, present := reflect.TypeOf(opts).Elem().FieldByName("Usage"); present {
		return field.Tag.Get("usage")
	}
	return ""
}

// A ByteSize is used for flags that represent some quantity of bytes that can be
// passed as human-readable quantities (eg. "10G").
type ByteSize uint64

// UnmarshalFlag implements the flags.Unmarshaler interface.
func (b *ByteSize) UnmarshalFlag(in string) error {
	b2, err := humanize.ParseBytes(in)
	*b = ByteSize(b2)
	return flagsError(err)
}

// UnmarshalText implements the encoding.TextUnmarshaler interface
func (b *ByteSize) UnmarshalText(text []byte) error {
	return b.UnmarshalFlag(string(text))
}

// A Duration is used for flags that represent a time duration; it's just a wrapper
// around time.Duration that implements the flags.Unmarshaler and
// encoding.TextUnmarshaler interfaces.
type Duration time.Duration

// UnmarshalFlag implements the flags.Unmarshaler interface.
func (d *Duration) UnmarshalFlag(in string) error {
	d2, err := time.ParseDuration(in)
	// For backwards compatibility, treat missing units as seconds.
	if err != nil {
		if d3, err := strconv.Atoi(in); err == nil {
			*d = Duration(time.Duration(d3) * time.Second)
			return nil
		}
	}
	*d = Duration(d2)
	return flagsError(err)
}

// UnmarshalText implements the encoding.TextUnmarshaler interface
func (d *Duration) UnmarshalText(text []byte) error {
	return d.UnmarshalFlag(string(text))
}

// A URL is used for flags or config fields that represent a URL.
// It's just a string because it's more convenient that way; we haven't needed them as a net.URL so far.
type URL string

// UnmarshalFlag implements the flags.Unmarshaler interface.
func (u *URL) UnmarshalFlag(in string) error {
	if _, err := url.Parse(in); err != nil {
		return flagsError(err)
	}
	*u = URL(in)
	return nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface
func (u *URL) UnmarshalText(text []byte) error {
	return u.UnmarshalFlag(string(text))
}

// String implements the fmt.Stringer interface
func (u *URL) String() string {
	return string(*u)
}

// A Version is an extension to semver.Version extending it with the ability to
// recognise >= prefixes.
type Version struct {
	semver.Version
	IsGTE bool
}

// UnmarshalText implements the encoding.TextUnmarshaler interface
func (v *Version) UnmarshalText(text []byte) error {
	return v.UnmarshalFlag(string(text))
}

// UnmarshalFlag implements the flags.Unmarshaler interface.
func (v *Version) UnmarshalFlag(in string) error {
	if strings.HasPrefix(in, ">=") {
		v.IsGTE = true
		in = strings.TrimSpace(strings.TrimPrefix(in, ">="))
	}
	return v.Set(in)
}

// String implements the fmt.Stringer interface
func (v Version) String() string {
	if v.IsGTE {
		return ">=" + v.Version.String()
	}
	return v.Version.String()
}

// VersionString returns just the version, without any preceding >=.
func (v *Version) VersionString() string {
	return v.Version.String()
}

// Semver converts a Version to a semver.Version
func (v *Version) Semver() semver.Version {
	return v.Version
}

// flagsError converts an error to a flags.Error, which is required for flag parsing.
func flagsError(err error) error {
	if err == nil {
		return err
	}
	return &flags.Error{Type: flags.ErrMarshal, Message: err.Error()}
}
