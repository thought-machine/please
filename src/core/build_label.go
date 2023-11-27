package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/thought-machine/go-flags"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/cmap"
	"github.com/thought-machine/please/src/process"
)

var log = logging.Log

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
	zero := BuildLabel{} //nolint:ifshort
	if label == zero {
		return ""
	} else if label == OriginalTarget {
		return "command-line targets"
	}
	s := "//" + label.PackageName
	if label.Subrepo != "" {
		s = "///" + label.Subrepo + s
	}
	if label.IsAllSubpackages() {
		if label.PackageName == "" {
			return s + "..."
		}
		return s + "/..."
	}
	return s + ":" + label.Name
}

// ShortString returns a string representation of this build label, abbreviated if
// possible, and relative to the given label.
func (label BuildLabel) ShortString(context BuildLabel) string {
	if label.Subrepo != context.Subrepo {
		return label.String()
	} else if label.PackageName == context.PackageName {
		return ":" + label.Name
	} else if label.Name == filepath.Base(label.PackageName) {
		return "//" + label.PackageName
	}
	label.Subrepo = ""
	return label.String()
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
	label, err := TryParseBuildLabel(target, currentPath, "")
	if err != nil {
		panic(err)
	}
	return label
}

// SplitLabelAnnotation splits the build label from the annotation
func SplitLabelAnnotation(target string) (string, string) {
	parts := strings.Split(target, "|")
	annotation := ""
	if len(parts) == 2 {
		annotation = parts[1]
	}
	return parts[0], annotation
}

// TryParseBuildLabel attempts to parse a single build label from a string. Returns an error if unsuccessful.
func TryParseBuildLabel(target, currentPath, subrepo string) (BuildLabel, error) {
	if pkg, name, subrepo := ParseBuildLabelParts(target, currentPath, subrepo); name != "" {
		return BuildLabel{PackageName: pkg, Name: name, Subrepo: subrepo}, nil
	}
	return BuildLabel{}, fmt.Errorf("Invalid build label: %s", target)
}

// SplitSubrepoArch splits a subrepo name into the subrepo and architecture parts
func SplitSubrepoArch(subrepoName string) (string, string) {
	if idx := strings.LastIndex(subrepoName, "@"); idx != -1 {
		return subrepoName[:idx], subrepoName[(idx + 1):]
	}
	return subrepoName, ""
}

// JoinSubrepoArch joins a subrepo name with an architecture
func JoinSubrepoArch(subrepoName, arch string) string {
	if subrepoName == "" {
		return arch
	}
	if arch == "" {
		return subrepoName
	}
	return fmt.Sprintf("%v@%v", subrepoName, arch)
}

// ParseBuildLabelParts parses a build label into the package & name parts.
// If valid, the name string will always be populated; the package string might not be if it's a local form.
func ParseBuildLabelParts(target, currentPath, subrepo string) (string, string, string) {
	if len(target) < 2 { // Always must start with // or : and must have at least one char following.
		return "", "", ""
	} else if target[0] == ':' {
		if !validateTargetName(target[1:]) {
			return "", "", ""
		}
		return currentPath, target[1:], ""
	} else if target[0] == '@' {
		// @subrepo//pkg:target or @subrepo:target syntax
		return parseBuildLabelSubrepo(target[1:], currentPath)
	} else if strings.HasPrefix(target, "///") {
		// ///subrepo/pkg:target syntax.
		return parseBuildLabelSubrepo(target[3:], currentPath)
	} else if target[0] != '/' || target[1] != '/' {
		return "", "", ""
	} else if idx := strings.IndexRune(target, ':'); idx != -1 {
		pkg := target[2:idx]
		name := target[idx+1:]
		// Check ... explicitly to prevent :... which isn't allowed.
		if !validatePackageName(pkg) || !validateTargetName(name) || name == "..." {
			return "", "", ""
		}
		return pkg, name, subrepo
	} else if !validatePackageName(target[2:]) {
		return "", "", ""
	}
	// Must be the abbreviated form (//pkg) or subtargets (//pkg/...), there's no : in it.
	if strings.HasSuffix(target, "/...") {
		return strings.TrimRight(target[2:len(target)-3], "/"), "...", ""
	} else if idx := strings.LastIndexByte(target, '/'); idx != -1 {
		return target[2:], target[idx+1:], subrepo
	}
	return target[2:], target[2:], subrepo
}

// parseBuildLabelSubrepo parses a build label that began with a subrepo symbol (either @ or ///).
func parseBuildLabelSubrepo(target, currentPath string) (string, string, string) {
	idx := strings.Index(target, "//")
	if idx == -1 {
		// if subrepo and target are the same name, then @subrepo syntax will also suffice
		if idx = strings.IndexByte(target, ':'); idx == -1 {
			if idx := strings.LastIndexByte(target, '/'); idx != -1 {
				return "", target[idx+1:], target
			}
			return "", target, target
		}
	}
	if strings.ContainsRune(target[:idx], ':') {
		return "", "", ""
	}
	pkg, name, _ := ParseBuildLabelParts(target[idx:], currentPath, "")
	return pkg, name, target[:idx]
}

// As above, but allows parsing of relative labels (eg. rules:python_rules)
// which is convenient at the shell prompt
func parseMaybeRelativeBuildLabel(target, subdir string) (BuildLabel, error) {
	// Try the ones that don't need locating the repo root first.
	startsWithColon := strings.HasPrefix(target, ":")
	if !startsWithColon {
		if !strings.HasPrefix(target, "//") && strings.HasPrefix(target, "/") {
			target = "/" + target
		}
		if label, err := TryParseBuildLabel(target, "", ""); err == nil || strings.HasPrefix(target, "//") {
			return label, err
		}
	}
	// Now we need to locate the repo root and initial package.
	// Deliberately leave this till after the above to facilitate the --repo_root flag.
	if subdir == "" {
		MustFindRepoRoot()
		subdir = InitialPackagePath
	}
	if startsWithColon {
		return TryParseBuildLabel(target, subdir, "")
	}
	// Presumably it's just underneath this directory (note that if it was absolute we returned above)
	return TryParseBuildLabel("//"+filepath.Join(subdir, target), "", "")
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

// IsPseudoTarget returns true if either the liable ends in ... or in all.
// It is useful to check if a liable potentially references more than one target.
func (label BuildLabel) IsPseudoTarget() bool {
	return label.IsAllSubpackages() || label.IsAllTargets()
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

// Compare compares this build label to another one (suitable for using with slices.SortFunc)
func (label BuildLabel) Compare(other BuildLabel) int {
	if label.Subrepo != other.Subrepo {
		if label.Subrepo < other.Subrepo {
			return -1
		}
		return 1
	} else if label.PackageName != other.PackageName {
		if label.PackageName < other.PackageName {
			return -1
		}
		return 1
	} else if label.Name != other.Name {
		if label.Name < other.Name {
			return -1
		}
		return 1
	}
	return 0
}

// Less returns true if this build label would sort less than another one.
func (label BuildLabel) Less(other BuildLabel) bool {
	if label.Subrepo != other.Subrepo {
		return label.Subrepo < other.Subrepo
	} else if label.PackageName != other.PackageName {
		return label.PackageName < other.PackageName
	}
	return label.Name < other.Name
}

// Paths is an implementation of BuildInput interface; we use build labels directly as inputs.
func (label BuildLabel) Paths(graph *BuildGraph) []string {
	target := graph.TargetOrDie(label)
	return addPathPrefix(target.Outputs(), target.PackageDir())
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
		ret[i] = filepath.Join(prefix, output)
	}
	return ret
}

// LocalPaths is an implementation of BuildInput interface.
func (label BuildLabel) LocalPaths(graph *BuildGraph) []string {
	return graph.TargetOrDie(label).Outputs()
}

// Label is an implementation of BuildInput interface. It always returns this label.
func (label BuildLabel) Label() (BuildLabel, bool) {
	return label, true
}

func (label BuildLabel) nonOutputLabel() (BuildLabel, bool) {
	return label, true
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
	l, err := TryParseBuildLabel(string(text), "", "")
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

// IsHidden return whether the target is an intermediate target created by the build definition.
func (label BuildLabel) IsHidden() bool {
	return label.Name[0] == '_'
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

// SubrepoLabel returns a build label corresponding to the subrepo part of this build label.
func (label BuildLabel) SubrepoLabel(state *BuildState) BuildLabel {
	subrepoName, arch := SplitSubrepoArch(label.Subrepo)
	if arch == "" && state.Arch != cli.HostArch() {
		arch = state.Arch.String()
	}

	plugin, ok := state.Config.Plugin[subrepoName]
	if !ok {
		return subrepoLabel(subrepoName, arch)
	}

	if plugin.Target.String() == "" {
		log.Fatalf("[Plugin \"%v\"] must have Target set in the .plzconfig", subrepoName)
	}

	t := plugin.Target
	// If the plugin already specifies an architecture, don't override it
	if strings.Contains(t.Subrepo, "@") {
		return t
	}

	// Otherwise we need to set it to match our architecture
	t.Subrepo = JoinSubrepoArch(t.Subrepo, arch)
	return t
}

func subrepoLabel(subrepoName, arch string) BuildLabel {
	if idx := strings.LastIndexByte(subrepoName, '/'); idx != -1 {
		return BuildLabel{PackageName: subrepoName[:idx], Name: subrepoName[idx+1:], Subrepo: arch}
	}
	// This is legit, the subrepo is defined at the root.
	return BuildLabel{Name: subrepoName, Subrepo: arch}
}

func hashBuildLabel(l BuildLabel) uint64 {
	return cmap.XXHashes(l.Subrepo, l.PackageName, l.Name)
}

// packageKey returns a key for this build label that only uses the subrepo and package parts.
func (label BuildLabel) packageKey() packageKey {
	return packageKey{Name: label.PackageName, Subrepo: label.Subrepo}
}

func hashPackageKey(key packageKey) uint64 {
	return cmap.XXHashes(key.Subrepo, key.Name)
}

// CanSee returns true if label can see the given dependency, or false if not.
func (label BuildLabel) CanSee(state *BuildState, dep *BuildTarget) bool {
	// Targets are always visible to other targets in the same directory.
	if label.PackageName == dep.Label.PackageName {
		return true
	} else if dep.Label.isExperimental(state) && !label.isExperimental(state) {
		log.Error("Target %s cannot depend on experimental target %s", label, dep.Label)
		return false
	}
	parent := label.Parent()
	for _, vis := range dep.Visibility {
		if vis.Includes(parent) {
			return true
		}
	}
	if dep.Label.PackageName == parent.PackageName {
		return true
	}
	if label.isExperimental(state) {
		log.Info("Visibility restrictions suppressed for %s since %s is in the experimental tree", dep.Label, label)
		return true
	}
	return false
}

// isExperimental returns true if this label is in the "experimental" tree
func (label BuildLabel) isExperimental(state *BuildState) bool {
	for _, exp := range state.experimentalLabels {
		if exp.Includes(label) {
			return true
		}
	}
	return false
}

// Matches returns whether the build label matches the other based on wildcard rules
func (label BuildLabel) Matches(other BuildLabel) bool {
	if label.Name == "..." {
		return label.PackageName == "." || strings.HasPrefix(other.PackageName, label.PackageName)
	}
	if label.Name == "all" {
		return label.PackageName == other.PackageName
	}
	// Allow //foo:_bar#bazz to match //foo:bar
	return label == other.Parent()
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
	out, _, err := process.New().ExecWithTimeout(context.Background(), nil, "", os.Environ(), 10*time.Second, false, false, false, false, process.NoSandbox, append([]string{exec}, os.Args[1:]...))
	if err != nil {
		return nil
	}
	var ret []flags.Completion
	for _, line := range strings.Split(string(out), "\n") {
		if line != "" {
			ret = append(ret, flags.Completion{Item: line, Description: "BuildLabel"})
		}
	}
	return ret
}

// MarshalText implements the encoding.TextMarshaler interface, which makes BuildLabels
// usable as map keys in JSON.
// This implementation never returns an error.
func (label BuildLabel) MarshalText() ([]byte, error) {
	return []byte(label.String()), nil
}

func (label BuildLabel) InSamePackageAs(l BuildLabel) bool {
	return l.PackageName == label.PackageName && l.Subrepo == label.Subrepo
}

// A packageKey is a cut-down version of BuildLabel that only contains the package part.
// It's used to key maps and so forth that don't care about the target name.
type packageKey struct {
	Name, Subrepo string
}

// String implements the traditional fmt.Stringer interface.
func (key packageKey) String() string {
	if key.Subrepo != "" {
		return "@" + key.Subrepo + "//" + key.Name
	}
	return key.Name
}

// BuildLabel returns a build label representing this package key.
func (key packageKey) BuildLabel() BuildLabel {
	return BuildLabel{
		Subrepo:     key.Subrepo,
		PackageName: key.Name,
		Name:        "all",
	}
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
func (slice BuildLabels) String() string {
	s := make([]string, len(slice))
	for i, l := range slice {
		s[i] = l.String()
	}
	return strings.Join(s, ", ")
}
