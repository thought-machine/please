package core

import (
	"path"
	"regexp"
	"strings"

	"fmt"
	"gopkg.in/op/go-logging.v1"
)

var log = logging.MustGetLogger("core")

// Representation of an identifier of a build target, eg. //spam/eggs:ham
// corresponds to BuildLabel{PackageName: spam/eggs name: ham}
// BuildLabels are always absolute, so relative identifiers
// like :ham are always parsed into an absolute form.
// There is also implicit expansion of the final element of a target (ala Blaze)
// so //spam/eggs is equivalent to //spam/eggs:eggs
type BuildLabel struct {
	PackageName string
	Name        string
}

// Build label that represents parsing the entire graph.
// We use this specially in one or two places.
var WholeGraph = []BuildLabel{{PackageName: "", Name: "..."}}

// Used to indicate that we're going to consume build labels from stdin.
var BuildLabelStdin = BuildLabel{PackageName: "", Name: "_STDIN"}

// Used to indicate one of the originally requested targets on the command line.
var OriginalTarget = BuildLabel{PackageName: "", Name: "_ORIGINAL"}

const validChars = `\pL\pN\pM!@`
const packagePart = "[" + validChars + `\._\+-]+`
const packageName = "(" + packagePart + "(?:/" + packagePart + ")*)"
const targetName = `([` + validChars + `_\+-][` + validChars + `\._\+-]*(?:#[` + validChars + `_\+-]+)*)`

// Regexes for matching the various ways of writing a build label.
// Fully specified labels, e.g. //src/core:core
var absoluteTarget = regexp.MustCompile(fmt.Sprintf("^//(?:%s)?:%s$", packageName, targetName))

// Targets in local package, e.g. :core
var localTarget = regexp.MustCompile(fmt.Sprintf("^:%s$", targetName))

// Targets with an implicit target name, e.g. //src/core (expands to //src/core:core)
var implicitTarget = regexp.MustCompile(fmt.Sprintf("^//(?:%s/)?(%s)$", packageName, packagePart))

// All targets underneath a package, e.g. //src/core/...
var subTargets = regexp.MustCompile(fmt.Sprintf("^//%s/(\\.\\.\\.)$", packageName))

// Sub targets immediately underneath the root; //...
var rootSubTargets = regexp.MustCompile(fmt.Sprintf("^(//)(\\.\\.\\.)$"))

// The following cases only apply on the command line and can't be used in BUILD files.
// A relative target, e.g. core:core (expands to //src/core:core if already in src)
var relativeTarget = regexp.MustCompile(fmt.Sprintf("^%s:%s$", packageName, targetName))

// A relative target with implicitly specified target name, e.g. src/core (expands to //src/core:core)
var relativeImplicitTarget = regexp.MustCompile(fmt.Sprintf("^(?:%s/)?(%s)$", packageName, packagePart))

// All targets underneath a relative package, e.g. src/core/...
var relativeSubTargets = regexp.MustCompile(fmt.Sprintf("^(?:%s/)?(\\.\\.\\.)$", packageName))

// Package and target names only, used for validation.
var packageNameOnly = regexp.MustCompile(fmt.Sprintf("^%s?$", packageName))
var targetNameOnly = regexp.MustCompile(fmt.Sprintf("^%s$", targetName))

func (label BuildLabel) String() string {
	if label.Name != "" {
		return "//" + label.PackageName + ":" + label.Name
	}
	return "//" + label.PackageName
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
	if !packageNameOnly.MatchString(pkgName) {
		return fmt.Errorf("Invalid package name: %s", pkgName)
	} else if !targetNameOnly.MatchString(name) {
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

// ParseBuildLabel parses a single build label from a string. Panics on failure.
func ParseBuildLabel(target, currentPath string) BuildLabel {
	label, err := TryParseBuildLabel(target, currentPath)
	if err != nil {
		panic(err)
	}
	return label
}

// newBuildLabel creates a new build label after parsing.
// It is only used internally because it skips some checks; we know those are valid because
// they already run implicitly as part of us parsing the label.
func newBuildLabel(pkgName, name string) (BuildLabel, error) {
	return BuildLabel{pkgName, name}, validateSuffixes(pkgName, name)
}

// TryParseBuildLabel attempts to parse a single build label from a string. Returns an error if unsuccessful.
func TryParseBuildLabel(target string, currentPath string) (BuildLabel, error) {
	matches := absoluteTarget.FindStringSubmatch(target)
	if matches != nil {
		return newBuildLabel(matches[1], matches[2])
	}
	matches = localTarget.FindStringSubmatch(target)
	if matches != nil {
		return newBuildLabel(currentPath, matches[1])
	}
	matches = subTargets.FindStringSubmatch(target)
	if matches != nil {
		return newBuildLabel(matches[1], matches[2])
	}
	matches = rootSubTargets.FindStringSubmatch(target)
	if matches != nil {
		return newBuildLabel("", matches[2])
	}
	matches = implicitTarget.FindStringSubmatch(target)
	if matches != nil {
		if matches[1] != "" {
			return newBuildLabel(matches[1]+"/"+matches[2], matches[2])
		}
		return newBuildLabel(matches[2], matches[2])
	}
	return BuildLabel{}, fmt.Errorf("Invalid build label: %s", target)
}

// As above, but allows parsing of relative labels (eg. src/parse/rules:python_rules)
// which is convenient at the shell prompt
func parseMaybeRelativeBuildLabel(target, subdir string) (BuildLabel, error) {
	// Try the ones that don't need locating the repo root first.
	if !strings.HasPrefix(target, ":") {
		if label, err := TryParseBuildLabel(target, ""); err == nil {
			return label, nil
		}
	}
	// Now we need to locate the repo root and initial package.
	// Deliberately leave this till after the above to facilitate the --repo_root flag.
	if subdir == "" {
		MustFindRepoRoot()
		subdir = initialPackage
	}
	matches := relativeTarget.FindStringSubmatch(target)
	if matches != nil {
		return newBuildLabel(path.Join(subdir, matches[1]), matches[2])
	}
	matches = relativeSubTargets.FindStringSubmatch(target)
	if matches != nil {
		return newBuildLabel(path.Join(subdir, matches[1]), matches[2])
	}
	matches = relativeImplicitTarget.FindStringSubmatch(target)
	if matches != nil {
		if matches[1] != "" {
			return newBuildLabel(path.Join(subdir, matches[1], matches[2]), matches[2])
		}
		return newBuildLabel(path.Join(subdir, matches[2]), matches[2])
	}
	matches = localTarget.FindStringSubmatch(target)
	if matches != nil {
		return newBuildLabel(subdir, matches[1])
	}
	return BuildLabel{}, fmt.Errorf("Invalid build target label: %s", target)
}

// Parse a bunch of build labels.
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

// Returns true if the label ends in ..., ie. it includes all subpackages.
func (label BuildLabel) IsAllSubpackages() bool {
	return label.Name == "..."
}

// Returns true if the label is the pseudo-label referring to all targets in this package.
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

func (this BuildLabel) Less(that BuildLabel) bool {
	if this.PackageName == that.PackageName {
		return this.Name < that.Name
	} else {
		return this.PackageName < that.PackageName
	}
}

// Implementation of BuildInput interface
func (label BuildLabel) Paths(graph *BuildGraph) []string {
	target := graph.TargetOrDie(label)
	outputs := target.Outputs()
	ret := make([]string, len(outputs), len(outputs))
	for i, output := range outputs {
		ret[i] = path.Join(label.PackageName, output)
	}
	return ret
}

func (label BuildLabel) FullPaths(graph *BuildGraph) []string {
	target := graph.TargetOrDie(label)
	outputs := target.Outputs()
	ret := make([]string, len(outputs), len(outputs))
	for i, output := range outputs {
		ret[i] = path.Join(target.OutDir(), output)
	}
	return ret
}

func (label BuildLabel) LocalPaths(graph *BuildGraph) []string {
	return graph.TargetOrDie(label).Outputs()
}

func (label BuildLabel) Label() *BuildLabel {
	return &label
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
		log.Fatalf("%s", err)
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

// LooksLikeABuildLabel returns true if the string appears to be a build label, false if not.
// Useful for cases like rule sources where sources can be a filename or a label.
func LooksLikeABuildLabel(str string) bool {
	return strings.HasPrefix(str, "//") || strings.HasPrefix(str, ":")
}

// Make slices of these guys sortable.
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
