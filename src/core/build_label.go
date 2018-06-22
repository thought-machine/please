package core

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/jessevdk/go-flags"
	"gopkg.in/op/go-logging.v1"
)

var log = logging.MustGetLogger("core")

// A BuildLabel is a representation of an identifier of a build target, e.g. //spam/eggs:ham
// corresponds to BuildLabel{PackageName: spam/eggs name: ham}
// BuildLabels are always absolute, so relative identifiers
// like :ham are always parsed into an absolute form.
// There is also implicit expansion of the final element of a target (ala Blaze)
// so //spam/eggs is equivalent to //spam/eggs:eggs
//
// It can also be in a subrepo, in which case the syntax is @subrepo//spam/eggs:ham.
type BuildLabel struct {
	PackageName string
	Name        string
	Subrepo     string
}

// WholeGraph represents parsing the entire graph (i.e. //...).
// We use this specially in one or two places.
var WholeGraph = []BuildLabel{{PackageName: "", Name: "..."}}

// BuildLabelStdin is used to indicate that we're going to consume build labels from stdin.
var BuildLabelStdin = BuildLabel{PackageName: "", Name: "_STDIN"}

// OriginalTarget is used to indicate one of the originally requested targets on the command line.
var OriginalTarget = BuildLabel{PackageName: "", Name: "_ORIGINAL"}

// String returns a string representation of this build label.
func (label BuildLabel) String() string {
	if label.Subrepo != "" {
		return "@" + "//" + label.PackageName + ":" + label.Name
	}
	return "//" + label.PackageName + ":" + label.Name
}

// NewBuildLabel constructs a new build label from the given components. Panics on failure.
func NewBuildLabel(pkgName, name string) BuildLabel {
	label, err := TryNewBuildLabel(pkgName, name)
	if err != nil {
		panic(err)
	}
	return label
}

// TryNewBuildLabel constructs a new build label from the given components.
func TryNewBuildLabel(pkgName, name string) (BuildLabel, error) {
	if err := validateNames(pkgName, name); err != nil {
		return BuildLabel{}, err
	}
	return BuildLabel{PackageName: pkgName, Name: name}, nil
}

// validateNames returns an error if the package name of target name isn't accepted.
func validateNames(pkgName, name string) error {
	if !validatePackageName(pkgName) {
		return fmt.Errorf("Invalid package name: %s", pkgName)
	} else if !validateTargetName(name) {
		return fmt.Errorf("Invalid target name: %s", name)
	} else if err := validateSuffixes(pkgName, name); err != nil {
		return err
	}
	return nil
}

// validateSuffixes checks that there are no invalid suffixes on the target name.
func validateSuffixes(pkgName, name string) error {
	if strings.HasSuffix(name, buildDirSuffix) ||
		strings.HasSuffix(name, testDirSuffix) ||
		strings.HasSuffix(pkgName, buildDirSuffix) ||
		strings.HasSuffix(pkgName, testDirSuffix) {

		return fmt.Errorf("._build and ._test are reserved suffixes")
	}
	return nil
}

// validatePackageName checks whether this string is a valid package name and returns true if so.
func validatePackageName(name string) bool {
	return name == "" || (name[0] != '/' && name[len(name)-1] != '/' && !strings.ContainsAny(name, `|$*?[]{}:()&\`) && !strings.Contains(name, "//"))
}

// validateTargetName checks whether this string is a valid target name and returns true if so.
func validateTargetName(name string) bool {
	return name != "" && !strings.ContainsAny(name, `|$*?[]{}:()&/\`) && (name[0] != '.' || name == "...") &&
		!strings.HasSuffix(name, buildDirSuffix) && !strings.HasSuffix(name, testDirSuffix)
}

// ParseBuildLabel parses a single build label from a string. Panics on failure.
func ParseBuildLabel(target, currentPath string) BuildLabel {
	label, err := TryParseBuildLabel(target, currentPath)
	if err != nil {
		panic(err)
	}
	return label
}

// TryParseBuildLabel attempts to parse a single build label from a string. Returns an error if unsuccessful.
func TryParseBuildLabel(target, currentPath string) (BuildLabel, error) {
	if pkg, name, _ := parseBuildLabelParts(target, currentPath, nil); name != "" {
		return BuildLabel{PackageName: pkg, Name: name}, nil
	}
	return BuildLabel{}, fmt.Errorf("Invalid build label: %s", target)
}

// ParseBuildLabelSubrepo parses a build label, and returns a separate indicator of the subrepo it was in.
// It panics on error.
func ParseBuildLabelSubrepo(target string, pkg *Package) (BuildLabel, string) {
	if p, name, subrepo := parseBuildLabelParts(target, pkg.Name, pkg.Subrepo); name != "" {
		return BuildLabel{PackageName: p, Name: name}, subrepo
	}
	// It's gonna fail, let this guy panic for us.
	return ParseBuildLabel(target, pkg.Name), ""
}

// parseBuildLabelParts parses a build label into the package & name parts.
// If valid, the name string will always be populated; the package string might not be if it's a local form.
func parseBuildLabelParts(target, currentPath string, subrepo *Subrepo) (string, string, string) {
	if len(target) < 2 { // Always must start with // or : and must have at least one char following.
		return "", "", ""
	} else if target[0] == ':' {
		if !validateTargetName(target[1:]) {
			return "", "", ""
		}
		return currentPath, target[1:], ""
	} else if target[0] == '@' {
		// @subrepo//pkg:target or @subrepo:target syntax
		idx := strings.Index(target, "//")
		if idx == -1 {
			if idx = strings.IndexRune(target, ':'); idx == -1 {
				return "", "", ""
			}
		}
		pkg, name, _ := parseBuildLabelParts(target[idx:], currentPath, subrepo)
		if pkg == "" && name == "" {
			return "", "", ""
		}
		s := target[1:idx]
		if subrepo == nil || !strings.HasPrefix(pkg, s) {
			pkg = path.Join(s, pkg) // Combine it to //subrepo/pkg:target
		}
		return pkg, name, s
	} else if target[0] != '/' || target[1] != '/' {
		return "", "", ""
	} else if idx := strings.IndexRune(target, ':'); idx != -1 {
		pkg := target[2:idx]
		name := target[idx+1:]
		// Check ... explicitly to prevent :... which isn't allowed.
		if !validatePackageName(pkg) || !validateTargetName(name) || name == "..." {
			return "", "", ""
		}
		return pkg, name, ""
	} else if !validatePackageName(target[2:]) {
		return "", "", ""
	}
	// Must be the abbreviated form (//pkg) or subtargets (//pkg/...), there's no : in it.
	if strings.HasSuffix(target, "/...") {
		return strings.TrimRight(target[2:len(target)-3], "/"), "...", ""
	} else if idx := strings.LastIndexByte(target, '/'); idx != -1 {
		return target[2:], target[idx+1:], ""
	}
	return target[2:], target[2:], ""
}

// As above, but allows parsing of relative labels (eg. src/parse/rules:python_rules)
// which is convenient at the shell prompt
func parseMaybeRelativeBuildLabel(target, subdir string) (BuildLabel, error) {
	// Try the ones that don't need locating the repo root first.
	startsWithColon := strings.HasPrefix(target, ":")
	if !startsWithColon {
		if label, err := TryParseBuildLabel(target, ""); err == nil || strings.HasPrefix(target, "//") {
			return label, err
		}
	}
	// Now we need to locate the repo root and initial package.
	// Deliberately leave this till after the above to facilitate the --repo_root flag.
	if subdir == "" {
		MustFindRepoRoot()
		subdir = initialPackage
	}
	if startsWithColon {
		return TryParseBuildLabel(target, subdir)
	}
	// Presumably it's just underneath this directory (note that if it was absolute we returned above)
	return TryParseBuildLabel("//"+path.Join(subdir, target), "")
}

// ParseBuildLabels parses a bunch of build labels from strings. It dies on failure.
// Relative labels are allowed since this is generally used at initialisation.
func ParseBuildLabels(targets []string) []BuildLabel {
	ret := make([]BuildLabel, len(targets))
	for i, target := range targets {
		if label, err := parseMaybeRelativeBuildLabel(target, ""); err != nil {
			log.Fatalf("%s", err)
		} else {
			ret[i] = label
		}
	}
	return ret
}

// IsAllSubpackages returns true if the label ends in ..., ie. it includes all subpackages.
func (label BuildLabel) IsAllSubpackages() bool {
	return label.Name == "..."
}

// IsAllTargets returns true if the label is the pseudo-label referring to all targets in this package.
func (label BuildLabel) IsAllTargets() bool {
	return label.Name == "all"
}

// Includes returns true if label includes the other label (//pkg:target1 is covered by //pkg:all etc).
func (label BuildLabel) Includes(that BuildLabel) bool {
	if (label.PackageName == "" && label.IsAllSubpackages()) ||
		that.PackageName == label.PackageName ||
		strings.HasPrefix(that.PackageName, label.PackageName+"/") {
		// We're in the same package or a subpackage of this visibility spec
		if label.IsAllSubpackages() {
			return true
		} else if label.PackageName == that.PackageName {
			if label.Name == that.Name || label.IsAllTargets() {
				return true
			}
		}
	}
	return false
}

// Less returns true if this build label would sort less than another one.
func (label BuildLabel) Less(other BuildLabel) bool {
	if label.PackageName == other.PackageName {
		return label.Name < other.Name
	}
	return label.PackageName < other.PackageName
}

// Paths is an implementation of BuildInput interface; we use build labels directly as inputs.
func (label BuildLabel) Paths(graph *BuildGraph) []string {
	return addPathPrefix(graph.TargetOrDie(label).Outputs(), label.PackageName)
}

// FullPaths is an implementation of BuildInput interface.
func (label BuildLabel) FullPaths(graph *BuildGraph) []string {
	target := graph.TargetOrDie(label)
	return addPathPrefix(target.Outputs(), target.OutDir())
}

// addPathPrefix adds a prefix to all the entries in a slice.
func addPathPrefix(paths []string, prefix string) []string {
	ret := make([]string, len(paths))
	for i, output := range paths {
		ret[i] = path.Join(prefix, output)
	}
	return ret
}

// LocalPaths is an implementation of BuildInput interface.
func (label BuildLabel) LocalPaths(graph *BuildGraph) []string {
	return graph.TargetOrDie(label).Outputs()
}

// Label is an implementation of BuildInput interface. It always returns this label.
func (label BuildLabel) Label() *BuildLabel {
	return &label
}

func (label BuildLabel) nonOutputLabel() *BuildLabel {
	return &label
}

// ForPackage converts this build label to one that's relative for the given package.
func (label BuildLabel) ForPackage(pkg *Package) BuildLabel {
	// TODO(peterebden): HasPrefix here is not super elegant. We should probably be able to avoid it
	//                   if we were more selective about calling this.
	if pkg.Subrepo != nil && !strings.HasPrefix(label.PackageName, pkg.Subrepo.Name) {
		return BuildLabel{PackageName: path.Join(pkg.Subrepo.Name, label.PackageName), Name: label.Name}
	}
	return label
}

// UnmarshalFlag unmarshals a build label from a command line flag. Implementation of flags.Unmarshaler interface.
func (label *BuildLabel) UnmarshalFlag(value string) error {
	// This is only allowable here, not in any other usage of build labels.
	if value == "-" {
		*label = BuildLabelStdin
		return nil
	} else if l, err := parseMaybeRelativeBuildLabel(value, ""); err != nil {
		// This has to be fatal because of the way we're using the flags package;
		// we lose incoming flags if we return errors.
		// But don't die in completion mode.
		if os.Getenv("PLZ_COMPLETE") == "" {
			log.Fatalf("%s", err)
		}
	} else {
		*label = l
	}
	return nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
// This is used by gcfg to unmarshal the config files.
func (label *BuildLabel) UnmarshalText(text []byte) error {
	l, err := TryParseBuildLabel(string(text), "")
	*label = l
	return err
}

// Parent returns what would be the parent of a build label, or the label itself if it's parentless.
// Note that there is not a concrete guarantee that the returned label exists in the build graph,
// and that the label returned is the ultimate ancestor (ie. not necessarily immediate parent).
func (label BuildLabel) Parent() BuildLabel {
	index := strings.IndexRune(label.Name, '#')
	if index == -1 || !strings.HasPrefix(label.Name, "_") {
		return label
	}
	label.Name = strings.TrimLeft(label.Name[:index], "_")
	return label
}

// HasParent returns true if the build label has a parent that's not itself.
func (label BuildLabel) HasParent() bool {
	return label.Parent() != label
}

// IsEmpty returns true if this is an empty build label, i.e. nothing's populated it yet.
func (label BuildLabel) IsEmpty() bool {
	return label.PackageName == "" && label.Name == ""
}

// PackageDir returns a path to the directory this target is in.
// This is equivalent to PackageName in all cases except when at the repo root, when this
// will return . instead. This is often easier to use in build rules.
func (label BuildLabel) PackageDir() string {
	if label.PackageName == "" {
		return "."
	}
	return label.PackageName
}

// Complete implements the flags.Completer interface, which is used for shell completion.
// Unfortunately it's rather awkward to handle here; we need to do a proper parse in order
// to find out what the possible build labels are, and we're not ready for that yet.
// Returning to main is also awkward since the flags haven't parsed properly; all in all
// it seems an easier (albeit inelegant) solution to start things over by re-execing ourselves.
func (label BuildLabel) Complete(match string) []flags.Completion {
	if match == "" {
		os.Exit(0)
	}
	os.Setenv("PLZ_COMPLETE", match)
	os.Unsetenv("GO_FLAGS_COMPLETION")
	exec, _ := os.Executable()
	out, _, err := ExecWithTimeout(nil, "", os.Environ(), 0, 0, false, false, append([]string{exec}, os.Args[1:]...))
	if err != nil {
		return nil
	}
	ret := []flags.Completion{}
	for _, line := range strings.Split(string(out), "\n") {
		if line != "" {
			ret = append(ret, flags.Completion{Item: line})
		}
	}
	return ret
}

// LooksLikeABuildLabel returns true if the string appears to be a build label, false if not.
// Useful for cases like rule sources where sources can be a filename or a label.
func LooksLikeABuildLabel(str string) bool {
	return strings.HasPrefix(str, "//") || strings.HasPrefix(str, ":") || (strings.HasPrefix(str, "@") && (strings.ContainsRune(str, ':') || strings.Contains(str, "//")))
}

// BuildLabels makes slices of build labels sortable.
type BuildLabels []BuildLabel

func (slice BuildLabels) Len() int {
	return len(slice)
}
func (slice BuildLabels) Less(i, j int) bool {
	return slice[i].Less(slice[j])
}
func (slice BuildLabels) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}
