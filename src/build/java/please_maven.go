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

	"output"
)

var log = logging.MustGetLogger("please_maven")

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

type pomXml struct {
	GroupId              string          `xml:"groupId"`
	ArtifactId           string          `xml:"artifactId"`
	Version              string          `xml:"version"`
	Dependencies         pomDependencies `xml:"dependencies"`
	DependencyManagement struct {
		Dependencies pomDependencies `xml:"dependencies"`
	} `xml:"dependencyManagement"`
	Properties struct {
		Property []struct {
			XMLName xml.Name
			Value   string `xml:",chardata"`
		} `xml:",any"`
	} `xml:"properties"`
	Licences struct {
		Licence []struct {
			Name string `xml:"name"`
		} `xml:"license"`
	} `xml:"licenses"`
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
		Latest  string `xml:"latest"`
		Release string `xml:"release"`
	} `xml:"versioning"`
	Group, Artifact string
}

func (metadata mavenMetadataXml) LatestVersion() string {
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

var opts struct {
	Repository string   `short:"r" long:"repository" description:"Location of Maven repo" default:"https://repo1.maven.org/maven2"`
	Verbosity  int      `short:"v" long:"verbose" default:"1" description:"Verbosity of output (higher number = more output, default 1 -> warnings and errors only)"`
	Exclude    []string `short:"e" long:"exclude" description:"Artifacts to exclude from download"`
	Indent     bool     `short:"i" long:"indent" description:"Indent stdout lines appropriately"`
	Optional   []string `short:"o" long:"optional" description:"Optional dependencies to fetch"`
	BuildRules bool     `short:"b" long:"build_rules" description:"Print individual maven_jar build rules for each artifact"`
	Args       struct {
		Package []string
	} `positional-args:"yes" required:"yes"`
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
	// Handle properties nonsense, because of course it doesn't work this out for us...
	properties := map[string]string{}
	for _, prop := range pom.Properties.Property {
		properties[prop.XMLName.Local] = prop.Value
	}
	// There are also some nonsense properties that aren't described by the above.
	properties["project.groupId"] = group
	properties["project.version"] = version
	// Arbitrarily, some pom files have this different structure with the extra "dependencyManagement" level.
	handleDependencies(pom.Dependencies, properties, group, artifact, version)
	handleDependencies(pom.DependencyManagement.Dependencies, properties, group, artifact, version)
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
		if dep.Scope == "test" {
			continue
		}
		if dep.Optional && !shouldFetchOptionalDep(dep.ArtifactId) {
			log.Debug("Not fetching optional dependency %s:%s", dep.GroupId, dep.ArtifactId)
			continue
		}
		dep.GroupId = replaceVariables(dep.GroupId, properties)
		dep.ArtifactId = replaceVariables(dep.ArtifactId, properties)
		// Not sure what this is about; httpclient seems to do this. It seems completely unhelpful but
		// no doubt there's some highly obscure case where Maven aficionados consider this useful.
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
			// things (e.g. netty) expect the latest. Possibly we should try the same one then
			// fall back to latest if it doesn't exist. This is easier but no doubt incorrect somewhere.
			if dep.GroupId == group {
				dep.Version = version
			} else {
				dep.Version = fetchMetadata(dep.GroupId, dep.ArtifactId).LatestVersion()
			}
		}
		licences := fetchLicences(dep.GroupId, dep.ArtifactId, dep.Version)
		if opts.Indent {
			fmt.Printf(strings.Repeat(" ", currentIndent))
		}
		// Print in forward order if we're not doing build rules
		if !opts.BuildRules {
			if len(licences) > 0 {
				fmt.Printf("%s:%s:%s:%s\n", dep.GroupId, dep.ArtifactId, dep.Version, strings.Join(licences, "|"))
			} else {
				fmt.Printf("%s:%s:%s\n", dep.GroupId, dep.ArtifactId, dep.Version)
			}
		}
		opts.Exclude = append(opts.Exclude, dep.ArtifactId) // Don't do this one again.
		// Recurse so we get all transitive dependencies
		currentIndent += 2
		pom := fetchAndParse(dep.GroupId, dep.ArtifactId, dep.Version)
		process(pom, dep.GroupId, dep.ArtifactId, dep.Version)
		currentIndent -= 2
	}
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

func buildPomUrl(group, artifact, version string) string {
	if version == "" {
		// This is kind of exciting - we just assume the latest version if one isn't available.
		// Not sure what we're really meant to do but I'm losing the will to live over all this so #yolo
		version = fetchMetadata(group, artifact).LatestVersion()
		log.Notice("Version not specified for %s:%s, decided to use %s", group, artifact, version)
	}
	slashGroup := strings.Replace(group, ".", "/", -1)
	return opts.Repository + "/" + slashGroup + "/" + artifact + "/" + version + "/" + artifact + "-" + version + ".pom"
}

// fetchOrDie fetches a URL and returns the content, dying if it can't be found.
func fetchOrDie(url string) []byte {
	log.Debug("Downloading %s...", url)
	response, err := http.Get(url)
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
	url := buildPomUrl(group, artifact, version)
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
	output.ParseFlagsOrDie("please_maven", &opts)
	output.InitLogging(opts.Verbosity, "", 0)
	for _, pkg := range opts.Args.Package {
		split := strings.Split(pkg, ":")
		if len(split) != 3 {
			log.Fatalf("Incorrect usage: argument %s must be in the form group:artifact:version\n", pkg)
		}
		pom := fetchAndParse(split[0], split[1], split[2])
		process(pom, split[0], split[1], split[2])
	}
}
