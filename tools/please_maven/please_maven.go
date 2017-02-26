// Tool to locate third-party Java dependencies on Maven Central.
// It doesn't actually fetch them (we just use curl for that) but instead
// is used to identify their transitive dependencies and report those back
// to build other rules from.

package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"text/template"

	"gopkg.in/op/go-logging.v1"

	"cli"
)

var log = logging.MustGetLogger("please_maven")

var client http.Client

var currentIndent = 0

var mavenJarTemplate = template.Must(template.New("maven_jar").Parse(`
maven_jar(
    name = '{{ .ArtifactId }}',
    id = '{{ .GroupId }}:{{ .ArtifactId }}:{{ .Version }}',
    hash = '',{{ if .Dependencies.Dependency }}
    deps = [
{{ range .Dependencies.Dependency }}        ':{{ .ArtifactId }}',
{{ end }}    ],{{ end }}
)
`))

type pomProperty struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

type pomXml struct {
	GroupId              string          `xml:"groupId"`
	ArtifactId           string          `xml:"artifactId"`
	Version              string          `xml:"version"`
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
	Parent struct {
		GroupId    string `xml:"groupId"`
		ArtifactId string `xml:"artifactId"`
		Version    string `xml:"version"`
	} `xml:"parent"`
	PropertiesMap map[string]string
}

type pomDependencies struct {
	Dependency []struct {
		GroupId    string `xml:"groupId"`
		ArtifactId string `xml:"artifactId"`
		Version    string `xml:"version"`
		Scope      string `xml:"scope"`
		Optional   bool   `xml:"optional"`
		// TODO(pebers): Handle exclusions here.
	} `xml:"dependency"`
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

// AddProperty adds a property (typically from a parent or wherever), without overwriting.
func (pom *pomXml) AddProperty(property pomProperty) {
	if _, present := pom.PropertiesMap[property.XMLName.Local]; !present {
		pom.PropertiesMap[property.XMLName.Local] = property.Value
		pom.Properties.Property = append(pom.Properties.Property, property)
	}
}

var opts = struct {
	Usage      string
	Repository string   `short:"r" long:"repository" description:"Location of Maven repo" default:"https://repo1.maven.org/maven2"`
	Verbosity  int      `short:"v" long:"verbose" default:"1" description:"Verbosity of output (higher number = more output, default 1 -> warnings and errors only)"`
	Exclude    []string `short:"e" long:"exclude" description:"Artifacts to exclude from download"`
	Indent     bool     `short:"i" long:"indent" description:"Indent stdout lines appropriately"`
	Optional   []string `short:"o" long:"optional" description:"Optional dependencies to fetch"`
	BuildRules bool     `short:"b" long:"build_rules" description:"Print individual maven_jar build rules for each artifact"`
	Args       struct {
		Package []string
	} `positional-args:"yes" required:"yes"`
}{
	Usage: `
please_maven is a tool shipped with Please that communicates with Maven repositories
to work out what files to download given a package spec.

Example usage:
please_maven io.grpc:grpc-all:1.1.2
> io.grpc:grpc-auth:1.1.2:src:BSD 3-Clause
> io.grpc:grpc-core:1.1.2:src:BSD 3-Clause
> ...
Its output is similarly in the common Maven artifact format which can be used to create
maven_jar rules in BUILD files. It also outputs some notes on whether sources are
available and what licence the package is under, if it can find it.

Note that it does not do complex cross-package dependency resolution and doesn't
necessarily support every aspect of Maven's pom.xml format, which is pretty hard
to fully grok. The goal is to provide a backend to Please's built-in maven_jars
rule to make adding dependencies easier.
`,
}

// Replaces a Maven variable in the given string.
func replaceVariables(s string, properties map[string]string) string {
	if strings.HasPrefix(s, "${") {
		if prop, present := properties[s[2:len(s)-1]]; !present {
			log.Fatalf("Failed property lookup %s: %s\n", s, properties)
		} else {
			return prop
		}
	}
	return s
}

// parse parses a downloaded pom.xml. This is of course less trivial than you would hope.
func parse(response []byte, group, artifact, version string) *pomXml {
	pom := &pomXml{}
	// This is an absolutely awful hack; we should use a proper decoder, but that seems
	// to be provoking a panic from the linker for reasons I don't fully understand right now.
	response = bytes.Replace(response, []byte("encoding=\"ISO-8859-1\""), []byte{}, -1)
	if err := xml.Unmarshal(response, pom); err != nil {
		log.Fatalf("Error parsing XML response: %s\n", err)
	}
	// Clean up strings in case they have spaces
	pom.GroupId = strings.TrimSpace(pom.GroupId)
	pom.ArtifactId = strings.TrimSpace(pom.ArtifactId)
	pom.Version = strings.TrimSpace(pom.Version)
	for i, licence := range pom.Licences.Licence {
		pom.Licences.Licence[i].Name = strings.TrimSpace(licence.Name)
	}
	// Handle properties nonsense, because of course it doesn't work this out for us...
	pom.PropertiesMap = map[string]string{}
	for _, prop := range pom.Properties.Property {
		pom.PropertiesMap[prop.XMLName.Local] = prop.Value
	}
	// There are also some properties that aren't described by the above - "project" is a bit magic.
	pom.PropertiesMap["groupId"] = group
	pom.PropertiesMap["artifactId"] = artifact
	pom.PropertiesMap["version"] = version
	pom.PropertiesMap["project.groupId"] = group
	pom.PropertiesMap["project.version"] = version
	if pom.Parent.ArtifactId != "" {
		// Must inherit variables from the parent.
		parent := fetchAndParse(pom.Parent.GroupId, pom.Parent.ArtifactId, pom.Parent.Version)
		for _, prop := range parent.Properties.Property {
			pom.AddProperty(prop)
		}
	}
	pom.Version = replaceVariables(pom.Version, pom.PropertiesMap)
	// Sanity check, but must happen after we resolve variables.
	if (pom.GroupId != "" && group != pom.GroupId) ||
		(pom.ArtifactId != "" && artifact != pom.ArtifactId) ||
		(pom.Version != "" && version != "" && version != pom.Version) {
		// These are a bit fiddly since inexplicably the fields are sometimes empty.
		log.Fatalf("Bad artifact: expected %s:%s:%s, got %s:%s:%s\n", group, artifact, version, pom.GroupId, pom.ArtifactId, pom.Version)
	}
	return pom
}

// process takes a downloaded and parsed pom.xml and prints details of dependencies to fetch.
func process(pom *pomXml, group, artifact, version string) {
	// Arbitrarily, some pom files have this different structure with the extra "dependencyManagement" level.
	handleDependencies(pom.Dependencies, pom.PropertiesMap, group, artifact, version)
	handleDependencies(pom.DependencyManagement.Dependencies, pom.PropertiesMap, group, artifact, version)
	if opts.BuildRules {
		if err := mavenJarTemplate.Execute(os.Stdout, pomXml{
			GroupId:    group,
			ArtifactId: artifact,
			Version:    version,
			Dependencies: pomDependencies{
				append(pom.Dependencies.Dependency, pom.DependencyManagement.Dependencies.Dependency...),
			},
		}); err != nil {
			log.Fatalf("Error executing template: %s", err)
		}
	}
}

func fetchLicences(group, artifact, version string) []string {
	// Unfortunately we have to make an extra request for the licences.
	pom := fetchAndParse(group, artifact, version)
	ret := make([]string, len(pom.Licences.Licence), len(pom.Licences.Licence))
	for i, licence := range pom.Licences.Licence {
		ret[i] = licence.Name
	}
	return ret
}

func shouldFetchOptionalDep(artifact string) bool {
	for _, option := range opts.Optional {
		if option == artifact {
			return true
		}
	}
	return false
}

func handleDependencies(deps pomDependencies, properties map[string]string, group, artifact, version string) {
	for _, dep := range deps.Dependency {
		// This is a bit of a hack; our build model doesn't distinguish these in the way Maven does.
		// TODO(pebers): Consider allowing specifying these to this tool to produce test-only deps.
		// Similarly system deps don't actually get fetched from Maven.
		if dep.Scope == "test" || dep.Scope == "system" {
			log.Debug("Not fetching %s:%s because of scope", dep.GroupId, dep.ArtifactId)
			continue
		}
		if dep.Optional && !shouldFetchOptionalDep(dep.ArtifactId) {
			log.Debug("Not fetching optional dependency %s:%s", dep.GroupId, dep.ArtifactId)
			continue
		}
		log.Debug("Fetching %s:%s:%s (depended on by %s:%s:%s)", dep.GroupId, dep.ArtifactId, dep.Version, group, artifact, version)
		dep.GroupId = replaceVariables(dep.GroupId, properties)
		dep.ArtifactId = replaceVariables(dep.ArtifactId, properties)
		// Not sure what this is about; httpclient seems to do this. It seems completely unhelpful but
		// no doubt there's some highly obscure case where it's considered useful.
		properties[dep.ArtifactId+".version"] = ""
		properties[strings.Replace(dep.ArtifactId, "-", ".", -1)+".version"] = ""
		dep.Version = strings.Trim(replaceVariables(dep.Version, properties), "[]")
		if strings.Contains(dep.Version, ",") {
			log.Fatalf("Can't do dependency mediation for %s:%s:%s", dep.GroupId, dep.ArtifactId, dep.Version)
		}
		if isExcluded(dep.ArtifactId) {
			continue
		}
		if dep.Version == "" {
			// Not 100% sure what the logic should really be here; for example, jacoco
			// seems to leave these underspecified and expects the same version, but other
			// things seem to expect the latest. Most likely it is some complex resolution
			// logic, but we'll take a stab at the same if the group matches and the same
			// version exists, otherwise we'll take the latest.
			metadata := fetchMetadata(dep.GroupId, dep.ArtifactId)
			if dep.GroupId == group && metadata.HasVersion(version) {
				dep.Version = version
			} else {
				dep.Version = metadata.LatestVersion()
			}
		}
		licences := fetchLicences(dep.GroupId, dep.ArtifactId, dep.Version)
		if opts.Indent {
			fmt.Printf(strings.Repeat(" ", currentIndent))
		}
		// Print in forward order if we're not doing build rules
		if !opts.BuildRules {
			s := hasSource(dep.GroupId, dep.ArtifactId, dep.Version)
			fmt.Printf("%s:%s:%s:%s", dep.GroupId, dep.ArtifactId, dep.Version, s)
			if len(licences) > 0 {
				fmt.Printf(":%s", strings.Join(licences, "|"))
			}
			fmt.Print("\n")
		}
		opts.Exclude = append(opts.Exclude, dep.ArtifactId) // Don't do this one again.
		// Recurse so we get all transitive dependencies
		currentIndent += 2
		pom := fetchAndParse(dep.GroupId, dep.ArtifactId, dep.Version)
		process(pom, dep.GroupId, dep.ArtifactId, dep.Version)
		currentIndent -= 2
	}
}

// hasSource returns a string describing whether the given target has sources or not.
func hasSource(group, artifact, version string) string {
	url := buildUrl(group, artifact, version, "-sources.jar")
	// Somewhat irritatingly it doesn't seem to work to send a HEAD or similar to determine
	// presence without downloading the whole shebang.
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatalf("Bad request: %s", err)
	}
	response, err := client.Do(req)
	if err != nil {
		log.Warning("Error finding sources: %s", err)
		return "no_src"
	}
	response.Body.Close()
	if response.StatusCode >= 400 {
		return "no_src"
	}
	return "src"
}

// isExcluded returns true if this artifact should be excluded from the download.
func isExcluded(artifact string) bool {
	for _, exclude := range opts.Exclude {
		if exclude == artifact {
			return true
		}
	}
	return false
}

func buildUrl(group, artifact, version, suffix string) string {
	if version == "" {
		// This is kind of exciting - we just assume the latest version if one isn't available.
		version = fetchMetadata(group, artifact).LatestVersion()
		log.Notice("Version not specified for %s:%s, decided to use %s", group, artifact, version)
	}
	slashGroup := strings.Replace(group, ".", "/", -1)
	return opts.Repository + "/" + slashGroup + "/" + artifact + "/" + version + "/" + artifact + "-" + version + suffix
}

// fetchOrDie fetches a URL and returns the content, dying if it can't be found.
func fetchOrDie(url string) []byte {
	log.Notice("Downloading %s...", url)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatalf("Bad request: %s", err)
	}
	response, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error downloading %s: %s\n", url, err)
	} else if response.StatusCode < 200 || response.StatusCode > 299 {
		log.Fatalf("Error downloading %s: %s\n", url, response.Status)
	}
	defer response.Body.Close()
	content, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("Error receiving response from %s: %s\n", url, err)
	}
	return content
}

var fetchAndParseMemo = map[string]*pomXml{}

// fetchAndParse combines fetch() and parse() and memoises the results so we don't repeat requests.
func fetchAndParse(group, artifact, version string) *pomXml {
	key := fmt.Sprintf("%s:%s:%s", group, artifact, version)
	if ret := fetchAndParseMemo[key]; ret != nil {
		return ret
	}
	url := buildUrl(group, artifact, version, ".pom")
	content := fetchOrDie(url)
	ret := parse(content, group, artifact, version)
	fetchAndParseMemo[key] = ret
	return ret
}

var fetchMetadataMemo = map[string]*mavenMetadataXml{}

// fetchMetadata finds the latest available version of a package when nothing else is specified.
// Also memoises because why not.
func fetchMetadata(group, artifact string) *mavenMetadataXml {
	slashGroup := strings.Replace(group, ".", "/", -1)
	url := opts.Repository + "/" + slashGroup + "/" + artifact + "/maven-metadata.xml"
	if ret := fetchMetadataMemo[url]; ret != nil {
		return ret
	}
	content := fetchOrDie(url)
	ret := &mavenMetadataXml{Group: group, Artifact: artifact}
	if err := xml.Unmarshal(content, ret); err != nil {
		log.Fatalf("Error parsing XML response: %s\n", err)
	}
	fetchMetadataMemo[url] = ret
	return ret
}

func main() {
	cli.ParseFlagsOrDie("please_maven", "5.5.0", &opts)
	cli.InitLogging(opts.Verbosity)
	for _, pkg := range opts.Args.Package {
		split := strings.Split(pkg, ":")
		if len(split) != 3 {
			log.Fatalf("Incorrect usage: argument %s must be in the form group:artifact:version\n", pkg)
		}
		pom := fetchAndParse(split[0], split[1], split[2])
		process(pom, split[0], split[1], split[2])
	}
}
