package maven

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/Workiva/go-datastructures/queue"
	"github.com/jessevdk/go-flags"
	"gopkg.in/op/go-logging.v1"
)

var log = logging.MustGetLogger("maven")

type unversioned struct {
	GroupId    string `xml:"groupId"`
	ArtifactId string `xml:"artifactId"`
}

type Artifact struct {
	unversioned
	// Raw version as found in XML
	Version string `xml:"version"`
	// A full-blown Maven version spec. If the version is not parseable (which is allowed
	// to happen :( ) then we just use Version to interpret it as a string.
	ParsedVersion Version
	isParent      bool
	// A "soft version", for dependencies that don't have one specified and we want to
	// provide a hint about what to do in that case.
	SoftVersion string
	// somewhat awkward, we use this to pass through excluded artifacts from above.
	Exclusions []Artifact `xml:"exclusions>exclusion"`
}

// GroupPath returns the group ID as a path.
func (a *Artifact) GroupPath() string {
	return strings.Replace(a.GroupId, ".", "/", -1)
}

// MetadataPath returns the path to the metadata XML file for this artifact.
func (a *Artifact) MetadataPath() string {
	return a.GroupPath() + "/" + a.ArtifactId + "/maven-metadata.xml"
}

// Path returns the path to an artifact that we'd download.
func (a *Artifact) Path(suffix string) string {
	return a.GroupPath() + "/" + a.ArtifactId + "/" + a.ParsedVersion.Path + "/" + a.ArtifactId + "-" + a.ParsedVersion.Path + suffix
}

// PomPath returns the path to the pom.xml for this artifact.
func (a *Artifact) PomPath() string {
	return a.Path(".pom")
}

// SourcePath returns the path to the sources jar for this artifact.
func (a *Artifact) SourcePath() string {
	return a.Path("-sources.jar")
}

// String prints this artifact as a Maven identifier (i.e. GroupId:ArtifactId:Version)
func (a Artifact) String() string {
	return a.GroupId + ":" + a.ArtifactId + ":" + a.ParsedVersion.Path
}

// FromId loads this artifact from a Maven id.
func (a *Artifact) FromId(id string) error {
	split := strings.Split(id, ":")
	if len(split) != 3 {
		return fmt.Errorf("Invalid Maven artifact id %s; must be in the form group:artifact:version", id)
	}
	a.GroupId = split[0]
	a.ArtifactId = split[1]
	a.Version = split[2]
	a.ParsedVersion.Unmarshal(a.Version)
	return nil
}

// SetVersion updates the version on this artifact.
func (a *Artifact) SetVersion(ver string) {
	a.ParsedVersion.Unmarshal(ver)
	a.Version = a.ParsedVersion.Path
}

// UnmarshalFlag implements the flags.Unmarshaler interface.
// This lets us use Artifact instances directly as flags.
func (a *Artifact) UnmarshalFlag(value string) error {
	if err := a.FromId(value); err != nil {
		return &flags.Error{Type: flags.ErrMarshal, Message: err.Error()}
	}
	return nil
}

// IsExcluded returns true if the given artifact is in this one's list of exclusions.
func (a *Artifact) IsExcluded(a2 *Artifact) bool {
	for _, excl := range a.Exclusions {
		if excl.GroupId == a2.GroupId && excl.ArtifactId == a2.ArtifactId {
			return true
		}
	}
	return false
}

type pomProperty struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

type pomXml struct {
	Artifact
	sync.Mutex
	OriginalArtifact     Artifact
	Dependencies         pomDependencies `xml:"dependencies"`
	DependencyManagement struct {
		Dependencies pomDependencies `xml:"dependencies"`
	} `xml:"dependencyManagement"`
	Properties struct {
		Property []pomProperty `xml:",any"`
	} `xml:"properties"`
	Licences struct {
		Licence []struct {
			Name string `xml:"name"`
		} `xml:"license"`
	} `xml:"licenses"`
	Parent        Artifact `xml:"parent"`
	PropertiesMap map[string]string
	Dependors     []*pomXml
	HasSources    bool
}

type pomDependency struct {
	Artifact
	Pom      *pomXml
	Dependor *pomXml
	Scope    string `xml:"scope"`
	Optional bool   `xml:"optional"`
}

type pomDependencies struct {
	Dependency []*pomDependency `xml:"dependency"`
}

type mavenMetadataXml struct {
	Version    string `xml:"version"`
	Versioning struct {
		Latest   string `xml:"latest"`
		Release  string `xml:"release"`
		Versions struct {
			Version []string `xml:"version"`
		} `xml:"versions"`
	} `xml:"versioning"`
	Group, Artifact string
}

// LatestVersion returns the latest available version of a package
func (metadata *mavenMetadataXml) LatestVersion() string {
	if metadata.Versioning.Release != "" {
		return metadata.Versioning.Release
	} else if metadata.Versioning.Latest != "" {
		log.Warning("No release version for %s:%s, using latest", metadata.Group, metadata.Artifact)
		return metadata.Versioning.Latest
	} else if metadata.Version != "" {
		log.Warning("No release version for %s:%s", metadata.Group, metadata.Artifact)
		return metadata.Version
	}
	log.Fatalf("Can't find a version for %s:%s", metadata.Group, metadata.Artifact)
	return ""
}

// HasVersion returns true if the given package has the specified version.
func (metadata *mavenMetadataXml) HasVersion(version string) bool {
	for _, v := range metadata.Versioning.Versions.Version {
		if v == version {
			return true
		}
	}
	return false
}

// Unmarshal reads a metadata object from raw XML. It dies on any error.
func (metadata *mavenMetadataXml) Unmarshal(content []byte) {
	if err := xml.Unmarshal(content, metadata); err != nil {
		log.Fatalf("Error parsing metadata XML: %s\n", err)
	}
}

// AddProperty adds a property (typically from a parent or wherever), without overwriting.
func (pom *pomXml) AddProperty(property pomProperty) {
	if _, present := pom.PropertiesMap[property.XMLName.Local]; !present {
		pom.PropertiesMap[property.XMLName.Local] = property.Value
		pom.Properties.Property = append(pom.Properties.Property, property)
	}
}

// replaceVariables a Maven variable in the given string.
func (pom *pomXml) replaceVariables(s string) string {
	if strings.HasPrefix(s, "${") {
		if prop, present := pom.PropertiesMap[s[2:len(s)-1]]; !present {
			log.Fatalf("Failed property lookup %s: %s\n", s, pom.PropertiesMap)
		} else {
			return prop
		}
	}
	return s
}

// Unmarshal parses a downloaded pom.xml. This is of course less trivial than you would hope.
func (pom *pomXml) Unmarshal(f *Fetch, response []byte) {
	pom.OriginalArtifact = pom.Artifact // Keep a copy of this for later
	decoder := xml.NewDecoder(bytes.NewReader(response))
	// This is not beautiful; it assumes all inputs are utf-8 compatible, essentially, in order to handle
	// ISO-8859-1 inputs. Possibly we should use a real conversion although it's a little unclear what the
	// suggested way of doing that or packages to use are.
	decoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) { return input, nil }
	if err := decoder.Decode(pom); err != nil {
		log.Fatalf("Error parsing XML response: %s\n", err)
	}
	// Clean up strings in case they have spaces
	pom.GroupId = strings.TrimSpace(pom.GroupId)
	pom.ArtifactId = strings.TrimSpace(pom.ArtifactId)
	pom.SetVersion(strings.TrimSpace(pom.Version))
	for i, licence := range pom.Licences.Licence {
		pom.Licences.Licence[i].Name = strings.TrimSpace(licence.Name)
	}
	// Handle properties nonsense, because of course it doesn't work this out for us...
	pom.PropertiesMap = map[string]string{}
	for _, prop := range pom.Properties.Property {
		pom.PropertiesMap[prop.XMLName.Local] = prop.Value
	}
	// There are also some properties that aren't described by the above - "project" is a bit magic.
	pom.PropertiesMap["groupId"] = pom.GroupId
	pom.PropertiesMap["artifactId"] = pom.ArtifactId
	pom.PropertiesMap["version"] = pom.Version
	pom.PropertiesMap["project.groupId"] = pom.GroupId
	pom.PropertiesMap["project.version"] = pom.Version
	if pom.Parent.ArtifactId != "" {
		if pom.Parent.GroupId == pom.GroupId && pom.Parent.ArtifactId == pom.ArtifactId {
			log.Fatalf("Circular dependency: %s:%s:%s specifies itself as its own parent", pom.GroupId, pom.ArtifactId, pom.Version)
		}
		// Must inherit variables from the parent.
		pom.Parent.isParent = true
		pom.Parent.ParsedVersion.Unmarshal(pom.Parent.Version)
		parent := f.Pom(&pom.Parent)
		for _, prop := range parent.Properties.Property {
			pom.AddProperty(prop)
		}
	}
	pom.Version = pom.replaceVariables(pom.Version)
	// Arbitrarily, some pom files have this different structure with the extra "dependencyManagement" level.
	pom.Dependencies.Dependency = append(pom.Dependencies.Dependency, pom.DependencyManagement.Dependencies.Dependency...)
	if !pom.isParent { // Don't fetch dependencies of parents, that just gets silly.
		pom.HasSources = f.HasSources(&pom.Artifact)
		for _, dep := range pom.Dependencies.Dependency {
			pom.handleDependency(f, dep)
		}
	}
}

func (pom *pomXml) handleDependency(f *Fetch, dep *pomDependency) {
	// This is a bit of a hack; our build model doesn't distinguish these in the way Maven does.
	// TODO(pebers): Consider allowing specifying these to this tool to produce test-only deps.
	// Similarly system deps don't actually get fetched from Maven.
	if dep.Scope == "test" || dep.Scope == "system" {
		log.Info("Not fetching %s:%s because of scope", dep.GroupId, dep.ArtifactId)
		return
	}
	if dep.Optional && !f.ShouldInclude(dep.ArtifactId) {
		log.Info("Not fetching optional dependency %s:%s", dep.GroupId, dep.ArtifactId)
		return
	}
	dep.GroupId = pom.replaceVariables(dep.GroupId)
	dep.ArtifactId = pom.replaceVariables(dep.ArtifactId)
	dep.SetVersion(pom.replaceVariables(dep.Version))
	dep.Dependor = pom
	if f.IsExcluded(dep.ArtifactId) {
		log.Info("Not fetching %s, is excluded by command-line parameter", dep.Artifact)
		return
	} else if pom.OriginalArtifact.IsExcluded(&dep.Artifact) {
		log.Info("Not fetching %s, is excluded by its parent", dep.Artifact)
		return
	}
	log.Info("Fetching %s (depended on by %s)", dep.Artifact, pom.Artifact)
	f.Resolver.Submit(dep)
}

func (dep *pomDependency) Resolve(f *Fetch) {
	if dep.Version == "" {
		// If no version is specified, we can take any version that we've already found.
		if pom := f.Resolver.Pom(&dep.Artifact); pom != nil {
			dep.Pom = pom
			return
		}

		// Not 100% sure what the logic should really be here; for example, jacoco
		// seems to leave these underspecified and expects the same version, but other
		// things seem to expect the latest. Most likely it is some complex resolution
		// logic, but we'll take a stab at the same if the group matches and the same
		// version exists, otherwise we'll take the latest.
		if metadata := f.Metadata(&dep.Artifact); dep.GroupId == dep.Dependor.GroupId && metadata.HasVersion(dep.Dependor.Version) {
			dep.SoftVersion = dep.Dependor.Version
		} else {
			dep.SoftVersion = metadata.LatestVersion()
		}
	}
	dep.Pom = f.Pom(&dep.Artifact)
	if dep.Dependor != nil {
		dep.Pom.Lock()
		defer dep.Pom.Unlock()
		dep.Pom.Dependors = append(dep.Pom.Dependors, dep.Dependor)
	}
}

// AllDependencies returns all the dependencies for this package.
func (pom *pomXml) AllDependencies() []*pomXml {
	deps := make([]*pomXml, 0, len(pom.Dependencies.Dependency))
	for _, dep := range pom.Dependencies.Dependency {
		if dep.Pom != nil {
			deps = append(deps, dep.Pom)
		}
	}
	return deps
}

// AllLicences returns all the licences for this package.
func (pom *pomXml) AllLicences() []string {
	licences := make([]string, len(pom.Licences.Licence))
	for i, licence := range pom.Licences.Licence {
		licences[i] = licence.Name
	}
	return licences
}

// Compare implements the queue.Item interface to define the order we resolve dependencies in.
func (dep *pomDependency) Compare(item queue.Item) int {
	dep2 := item.(*pomDependency)
	// Primarily we order by versions; if the version is not given, it comes after one that does.
	if dep.Version == "" && dep2.Version != "" {
		return 1
	} else if dep.Version != "" && dep2.Version == "" {
		return -1
	}
	id1 := dep.String()
	id2 := dep2.String()
	if id1 < id2 {
		return -1
	} else if id1 > id2 {
		return 1
	}
	return 0
}

// A Version is a Maven version spec (see https://docs.oracle.com/middleware/1212/core/MAVEN/maven_version.htm),
// including range reference info (https://docs.oracle.com/middleware/1212/core/MAVEN/maven_version.htm)
// The above is pretty light on detail unfortunately (like how do you know the difference between a BuildNumber
// and a Qualifier) so we really are taking a bit of a guess here.
// If only semver had existed back then...
//
// Note that we don't (yet?) support broken ranges like (,1.0],[1.2,).
type Version struct {
	Min, Max  VersionPart
	Raw, Path string
	Parsed    bool
}

// A VersionPart forms part of a Version; it can be either an upper or lower bound.
type VersionPart struct {
	Qualifier                 string
	Major, Minor, Incremental int
	Inclusive                 bool
	Set                       bool
}

// Unmarshal parses a Version from a raw string.
// Errors are not reported since literally anything can appear in a Maven version specifier;
// an input like "thirty-five ham and cheese sandwiches" is simply treated as a string.
func (v *Version) Unmarshal(in string) {
	v.Raw = in                      // Always.
	v.Path = strings.Trim(in, "[]") // needs more thought.
	// Try to match the simple single versions first.
	if submatches := singleVersionRegex.FindStringSubmatch(in); len(submatches) == 7 {
		// Special case for no specifiers; that indicates >=
		if submatches[1] == "[" || (submatches[1] == "" && submatches[6] == "") {
			v.Min = versionPart(submatches[2:6], true)
			v.Max.Major = 9999 // arbitrarily large
			v.Parsed = true
		}
		if submatches[6] == "]" {
			v.Max = versionPart(submatches[2:6], true)
			v.Parsed = true
		}
	} else if submatches := doubleVersionRegex.FindStringSubmatch(in); len(submatches) == 11 {
		v.Min = versionPart(submatches[2:6], submatches[1] == "[")
		v.Max = versionPart(submatches[6:10], submatches[10] == "]")
		v.Parsed = true
	}
}

// Matches returns true if this version matches the spec given by ver.
// Note that this is not symmetric; if this version is 1.0 and ver is <= 2.0, this is true;
// conversely it is false if this is 2.0 and ver is <= 1.0.
// It further treats this version as exact using its Min attribute, since that's roughly how Maven does it.
func (v *Version) Matches(ver *Version) bool {
	if v.Parsed || ver.Parsed {
		return (v.Min.LessThan(ver.Max) && v.Min.GreaterThan(ver.Min))
	}
	// If we fail to parse it, they are treated as strings.
	return v.Raw >= ver.Raw
}

// LessThan returns true if this version is less than the given version.
func (v *Version) LessThan(ver *Version) bool {
	if v.Parsed {
		return v.Min.LessThan(ver.Min)
	}
	return v.Raw < ver.Raw
}

// Intersect reduces v to the intersection of itself and v2.
// It returns true if the resulting version is still conceptually satisfiable.
func (v *Version) Intersect(v2 *Version) bool {
	if !v.Parsed || !v2.Parsed {
		// Fallback logic; one or both versions aren't parsed, so we do string comparisons.
		if strings.HasPrefix(v.Raw, "[") || strings.HasPrefix(v2.Raw, "[") {
			return false // No intersection if one specified an exact version
		} else if v2.Raw > v.Raw {
			*v = *v2
		}
		return true // still OK, we take the highest of the two.
	}
	if v2.Min.Set && v2.Min.GreaterThan(v.Min) {
		v.Min = v2.Min
	}
	if v2.Max.Set && v2.Max.LessThan(v.Max) {
		v.Max = v2.Max
	}
	return v.Min.LessThan(v.Max)
}

// Equals returns true if the two versions are equal.
func (v1 VersionPart) Equals(v2 VersionPart) bool {
	return v1.Major == v2.Major && v1.Minor == v2.Minor && v1.Incremental == v2.Incremental && v1.Qualifier == v2.Qualifier
}

// LessThan returns true if v1 < v2 (or <= if v2.Inclusive)
func (v1 VersionPart) LessThan(v2 VersionPart) bool {
	return v1.Major < v2.Major ||
		(v1.Major == v2.Major && v1.Minor < v2.Minor) ||
		(v1.Major == v2.Major && v1.Minor == v2.Minor && v1.Incremental < v2.Incremental) ||
		(v1.Major == v2.Major && v1.Minor == v2.Minor && v1.Incremental == v2.Incremental && v1.Qualifier < v2.Qualifier) ||
		(v2.Inclusive && v1.Equals(v2))
}

// GreaterThan returns true if v1 > v2 (or >= if v2.Inclusive)
func (v1 VersionPart) GreaterThan(v2 VersionPart) bool {
	return v1.Major > v2.Major ||
		(v1.Major == v2.Major && v1.Minor > v2.Minor) ||
		(v1.Major == v2.Major && v1.Minor == v2.Minor && v1.Incremental > v2.Incremental) ||
		(v1.Major == v2.Major && v1.Minor == v2.Minor && v1.Incremental == v2.Incremental && v1.Qualifier > v2.Qualifier) ||
		(v2.Inclusive && v1.Equals(v2))
}

// versionPart returns a new VersionPart given some raw strings.
func versionPart(parts []string, inclusive bool) VersionPart {
	v := VersionPart{
		Major:     mustInt(parts[0]),
		Qualifier: parts[3],
		Inclusive: inclusive,
		Set:       true,
	}
	if parts[1] != "" {
		v.Minor = mustInt(parts[1])
	}
	if parts[2] != "" {
		v.Incremental = mustInt(parts[2])
	}
	return v
}

func mustInt(in string) int {
	i, err := strconv.Atoi(in)
	if err != nil {
		log.Fatalf("Bad version number: %s", err)
	}
	return i
}

const versionRegex = `([0-9]+)(?:\.([0-9]+))?(?:\.([0-9]+))?(-[^\]\]]+)?`

var singleVersionRegex = regexp.MustCompile(fmt.Sprintf(`^(\[|\(,)?%s(\]|,\))?$`, versionRegex))
var doubleVersionRegex = regexp.MustCompile(fmt.Sprintf(`^(\[|\()%s,%s(\]|\))$`, versionRegex, versionRegex))
