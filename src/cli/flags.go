// Package cli contains helper functions related to flag parsing and logging.
package cli

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/coreos/go-semver/semver"
	"github.com/dustin/go-humanize"
	cli "github.com/peterebden/go-cli-init/v5/flags"
	clilogging "github.com/peterebden/go-cli-init/v5/logging"
	"github.com/thought-machine/go-flags"
)

// GiByte is a re-export for convenience of other things using it.
const GiByte = humanize.GiByte

// MinVerbosity is the minimum verbosity we support.
const MinVerbosity = clilogging.MinVerbosity

// MaxVerbosity is the maximum verbosity we support.
const MaxVerbosity = clilogging.MaxVerbosity

// ParseFlagsOrDie parses the app's flags and dies if unsuccessful.
// Also dies if any unexpected arguments are passed.
// It returns the active command if there is one.
func ParseFlagsOrDie(appname string, data interface{}) string {
	return cli.ParseFlagsOrDie(appname, data, nil)
}

// ParseFlagsFromArgsOrDie is similar to ParseFlagsOrDie but allows control over the
// flags passed.
// It returns the active command if there is one.
func ParseFlagsFromArgsOrDie(appname string, data interface{}, args []string, additionalUsageInfo cli.AdditionalUsageInfo) string {
	return cli.ParseFlagsFromArgsOrDie(appname, data, args, additionalUsageInfo)
}

// ParseFlags parses the app's flags and returns the parser, any extra arguments, and any error encountered.
// It may exit if certain options are encountered (eg. --help).
func ParseFlags(appname string, data interface{}, args []string, opts flags.Options, completionHandler cli.CompletionHandler, additionalUsageInfo cli.AdditionalUsageInfo) (*flags.Parser, []string, error) {
	return cli.ParseFlags(appname, data, args, opts, completionHandler, additionalUsageInfo)
}

// PrintCompletions prints a set of completions to stdout.
func PrintCompletions(items []flags.Completion) {
	for _, item := range items {
		fmt.Println(item.Item)
	}
}

// ActiveCommand returns the name of the currently active command.
func ActiveCommand(command *flags.Command) string {
	return cli.ActiveCommand(command)
}

// ActiveFullCommand returns the full name of the currently active command.
func ActiveFullCommand(command *flags.Command) string {
	return cli.ActiveFullCommand(command)
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
type Duration = cli.Duration

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
func (u URL) String() string {
	return string(u)
}

// AsURL returns this as a url.URL
// It is assumed never to fail because this URL has already been successfully parsed, at which
// point it is checked for validity.
func (u URL) AsURL() *url.URL {
	ret, _ := url.Parse(string(u))
	return ret
}

// A Version is an extension to semver.Version extending it with the ability to
// recognise >= prefixes.
type Version struct {
	semver.Version
	IsGTE bool
	IsSet bool
}

// NewVersion creates a new version from the given string.
func NewVersion(in string) (*Version, error) {
	v := &Version{}
	return v, v.UnmarshalFlag(in)
}

// MustNewVersion creates a new version and dies if it is not parseable.
func MustNewVersion(in string) *Version {
	v, err := NewVersion(in)
	if err != nil {
		log.Fatalf("Failed to parse version: %s", in)
	}
	return v
}

// MarshalText implements the encoding.TextMarshaler interface.
func (v Version) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
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
	v.IsSet = true
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

// Unset resets this version to the default.
func (v *Version) Unset() {
	*v = Version{}
}

// flagsError converts an error to a flags.Error, which is required for flag parsing.
func flagsError(err error) error {
	if err == nil {
		return nil
	}
	return &flags.Error{Type: flags.ErrMarshal, Message: err.Error()}
}

// A Filepath implements completion for file paths.
// This is distinct from upstream's in that it knows about completing into directories.
type Filepath string

// Complete implements the flags.Completer interface.
func (f *Filepath) Complete(match string) []flags.Completion {
	matches, _ := filepath.Glob(match + "*")
	// If there's exactly one match and it's a directory, take its contents instead.
	if len(matches) == 1 {
		if info, err := os.Stat(matches[0]); err == nil && info.IsDir() {
			matches, _ = filepath.Glob(matches[0] + "/*")
		}
	}
	ret := make([]flags.Completion, len(matches))
	for i, match := range matches {
		ret[i].Item = match
	}
	return ret
}

// Filepaths is a convenience type that is a list of file paths that knows how to convert itself to strings.
type Filepaths []Filepath

// AsStrings returns this slice of filepaths as a slice of strings.
func (f Filepaths) AsStrings() []string {
	ret := make([]string, len(f))
	for i, fp := range f {
		ret[i] = string(fp)
	}
	return ret
}

// Arch represents a combined Go-style operating system and architecture pair, as in "linux_amd64".
type Arch struct {
	OS, Arch string
}

// NewArch constructs a new Arch instance.
func NewArch(os, arch string) Arch {
	return Arch{OS: os, Arch: arch}
}

func NewArchFromString(arch string) Arch {
	parts := strings.Split(arch, "_")
	return Arch{OS: parts[0], Arch: parts[1]}
}

// HostArch returns the architecture for the host OS.
func HostArch() Arch {
	return Arch{OS: runtime.GOOS, Arch: runtime.GOARCH}
}

// String prints this Arch to its string representation.
func (arch *Arch) String() string {
	return arch.OS + "_" + arch.Arch
}

// MarshalText implements the encoding.TextMarshaler interface.
func (arch Arch) MarshalText() ([]byte, error) {
	return []byte(arch.String()), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface
func (arch *Arch) UnmarshalText(text []byte) error {
	return arch.UnmarshalFlag(string(text))
}

// UnmarshalFlag implements the flags.Unmarshaler interface.
func (arch *Arch) UnmarshalFlag(in string) error {
	if parts := strings.Split(in, "_"); len(parts) == 2 && !strings.ContainsRune(in, '/') && !strings.Contains(in, "@") {
		arch.OS = parts[0]
		arch.Arch = parts[1]
		return nil
	}
	return fmt.Errorf("Can't parse architecture %s (should be a Go-style arch pair, like 'linux_amd64' etc)", in)
}

// HostOS returns the OS of the host (machine doing the building).
// Configuring certain tools (e.g. pip) requires this information, even when cross-compiling.
func (arch *Arch) HostOS() string {
	return runtime.GOOS
}

// HostArch returns the architecture of the host (machine doing the building).
func (arch *Arch) HostArch() string {
	return runtime.GOARCH
}

// XOS returns the "alternative" OS spelling which some things prefer.
// The difference here is that "darwin" is instead returned as "osx".
func (arch *Arch) XOS() string {
	if arch.OS == "darwin" {
		return "osx"
	}
	return arch.OS
}

// XArch returns the "alternative" architecture spelling which some things prefer.
// In this case amd64 is instead returned as x86_64 and x86 as x86_32.
func (arch *Arch) XArch() string {
	switch arch.Arch {
	case "amd64":
		return "x86_64"
	case "x86":
		return "x86_32"
	case "arm64":
		return "aarch_64"
	default:
		return arch.Arch
	}
}

// GoArch returns the architecture as Go would name it.
func (arch *Arch) GoArch() string {
	switch arch.Arch {
	case "x86":
		return "386"
	case "x86-64":
		return "amd64"
	default:
		return arch.Arch
	}
}

// ContainsString returns true if the given slice contains an individual string.
func ContainsString(needle string, haystack []string) bool {
	return cli.ContainsString(needle, haystack)
}

// StdinStrings is an alias to the cli-init package.
type StdinStrings = cli.StdinStrings
