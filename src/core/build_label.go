package core

import (
	"path"
	"regexp"
	"strings"

	"fmt"
	"github.com/op/go-logging"
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

// This is a little strict; doesn't allow for non-ascii names, for example.
var absoluteTarget = regexp.MustCompile("^//([A-Za-z0-9\\._/-]*):([A-Za-z0-9\\._#+-]+)$")
var localTarget = regexp.MustCompile("^:([A-Za-z0-9\\._#+-]+)$")
var implicitTarget = regexp.MustCompile("^//([A-Za-z0-9\\._/-]+/)?([A-Za-z0-9\\._-]+)$")
var subTargets = regexp.MustCompile("^//([A-Za-z0-9\\._/-]+)/(\\.\\.\\.)$")
var rootSubTargets = regexp.MustCompile("^(//)(\\.\\.\\.)$")
var relativeTarget = regexp.MustCompile("^([A-Za-z0-9\\._-][A-Za-z0-9\\._/-]*):([A-Za-z0-9\\._#+-]+)$")
var relativeImplicitTarget = regexp.MustCompile("^([A-Za-z0-9\\._-][A-Za-z0-9\\._-]*/)([A-Za-z0-9\\._-]+)$")
var relativeSubTargets = regexp.MustCompile("^(?:([A-Za-z0-9\\._-][A-Za-z0-9\\._/-]*)/)?(\\.\\.\\.)$")

func (label BuildLabel) String() string {
	if label.Name != "" {
		return "//" + label.PackageName + ":" + label.Name
	}
	return "//" + label.PackageName
}

func NewBuildLabel(pkgName string, name string) BuildLabel {
	if strings.HasSuffix(pkgName, "/") || strings.HasSuffix(pkgName, ":") {
		panic("Invalid package name: " + pkgName)
	}
	return BuildLabel{PackageName: pkgName, Name: name}
}

// ParseBuildLabel parses a single build label from a string. Panics on failure.
func ParseBuildLabel(target string, currentPath string) BuildLabel {
	if label, success := tryParseBuildLabel(target, currentPath); success {
		return label
	}
	panic("Invalid build target label: " + target)
}

// ParseBuildFileLabel parses a build label with an attached file specification. Panics on failure.
func ParseBuildFileLabel(target string, currentPath string) (BuildLabel, string) {
	if strings.Count(target, ":") == 2 {
		index := strings.LastIndex(target, ":")
		return ParseBuildLabel(target[0:index], currentPath), target[index+1 : len(target)]
	}
	return ParseBuildLabel(target, currentPath), ""
}

// tryParseBuildLabel attempts to parse a single build label from a string. Returns a bool indicating success.
func tryParseBuildLabel(target string, currentPath string) (BuildLabel, bool) {
	matches := absoluteTarget.FindStringSubmatch(target)
	if matches != nil {
		return NewBuildLabel(matches[1], matches[2]), true
	}
	matches = localTarget.FindStringSubmatch(target)
	if matches != nil {
		return NewBuildLabel(currentPath, matches[1]), true
	}
	matches = subTargets.FindStringSubmatch(target)
	if matches != nil {
		return NewBuildLabel(matches[1], matches[2]), true
	}
	matches = rootSubTargets.FindStringSubmatch(target)
	if matches != nil {
		return NewBuildLabel("", matches[2]), true
	}
	matches = implicitTarget.FindStringSubmatch(target)
	if matches != nil {
		return NewBuildLabel(matches[1]+matches[2], matches[2]), true
	}
	return BuildLabel{}, false
}

// As above, but allows parsing of relative labels (eg. src/parse/rules:python_rules)
// which is convenient at the shell prompt
func parseMaybeRelativeBuildLabel(target, subdir string) BuildLabel {
	// Try the ones that don't need locating the repo root first.
	if !strings.HasPrefix(target, ":") {
		if label, success := tryParseBuildLabel(target, ""); success {
			return label
		}
	}
	// Now we need to locate the repo root and initial package.
	// Deliberately leave this till after the above to facilitate the --repo_root flag.
	if subdir == "" {
		_, subdir = getRepoRoot(true)
	}
	matches := relativeTarget.FindStringSubmatch(target)
	if matches != nil {
		return NewBuildLabel(path.Join(subdir, matches[1]), matches[2])
	}
	matches = relativeSubTargets.FindStringSubmatch(target)
	if matches != nil {
		return NewBuildLabel(path.Join(subdir, matches[1]), matches[2])
	}
	matches = relativeImplicitTarget.FindStringSubmatch(target)
	if matches != nil {
		return NewBuildLabel(path.Join(subdir, matches[1]+matches[2]), matches[2])
	}
	matches = localTarget.FindStringSubmatch(target)
	if matches != nil {
		return NewBuildLabel(subdir, matches[1])
	}
	log.Fatalf("Invalid build target label: %s", target)
	return BuildLabel{}
}

// Parse a bunch of build labels.
// Relative labels are allowed since this is generally used at initialisation.
func ParseBuildLabels(targets []string) []BuildLabel {
	defer func() {
		if r := recover(); r != nil {
			log.Fatalf("%s", r)
		}
	}()

	var ret []BuildLabel
	for _, target := range targets {
		ret = append(ret, parseMaybeRelativeBuildLabel(target, ""))
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

func (this BuildLabel) Less(that BuildLabel) bool {
	if this.PackageName == that.PackageName {
		return this.Name < that.Name
	} else {
		return this.PackageName < that.PackageName
	}
}

// Implementation of BuildInput interface
func (label BuildLabel) Paths(graph *BuildGraph) []string {
	target := graph.Target(label)
	if target == nil {
		panic(fmt.Sprintf("Couldn't find target %s in build graph", label))
	}
	ret := make([]string, len(target.Outputs()), len(target.Outputs()))
	for i, output := range target.Outputs() {
		ret[i] = path.Join(label.PackageName, output)
	}
	return ret
}

func (label BuildLabel) FullPaths(graph *BuildGraph) []string {
	target := graph.Target(label)
	if target == nil {
		panic("Couldn't find target corresponding to build label " + label.String())
	}
	ret := make([]string, len(target.Outputs()), len(target.Outputs()))
	for i, output := range target.Outputs() {
		ret[i] = path.Join(target.OutDir(), output)
	}
	return ret
}

func (label BuildLabel) Label() *BuildLabel {
	return &label
}

// Unmarshals a build label from a command line flag. Implementation of flags.Unmarshaler interface.
func (label *BuildLabel) UnmarshalFlag(value string) error {
	// This is only allowable here, not in any other usage of build labels.
	if value == "-" {
		*label = BuildLabelStdin
		return nil
	}

	defer func() {
		if r := recover(); r != nil {
			// This has to be fatal because of the way we're using the flags package;
			// we lose incoming flags if we return errors.
			log.Fatalf("%s", r)
		}
	}()
	*label = parseMaybeRelativeBuildLabel(value, "")
	return nil
}

// Returns true if the string appears to be a build label, false if not.
// Useful for cases like rule sources where sources can be a filename or a label.
func LooksLikeABuildLabel(str string) bool {
	return strings.HasPrefix(str, "/") || strings.HasPrefix(str, ":")
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
