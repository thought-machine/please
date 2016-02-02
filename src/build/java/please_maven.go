// Tool to locate third-party Java dependencies on Maven Central.
// It doesn't actually fetch them (we just use curl for that) but instead
// is used to identify their transitive dependencies and report those back
// to build other rules from.

package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/op/go-logging"

	"output"
)

var log = logging.MustGetLogger("please_maven")

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
		// TODO(pebers): Handle exclusions here.
	} `xml:"dependency"`
}

type mavenMetadataXml struct {
	Versioning struct {
		Latest  string `xml:"latest"`
		Release string `xml:"release"`
	} `xml:"versioning"`
}

var opts struct {
	Repository string   `short:"r" long:"repository" description:"Location of Maven repo" default:"https://repo1.maven.org/maven2"`
	Verbosity  int      `short:"v" long:"verbose" default:"1" description:"Verbosity of output (higher number = more output, default 1 -> warnings and errors only)"`
	Exclude    []string `short:"e" long:"exclude" description:"Artifacts to exclude from download"`
	Args       struct {
		Package []string
	} `positional-args:"yes" required:"yes"`
}

// Replaces a Maven variable in the given string.
func replaceVariables(s string, properties map[string]string) string {
	if strings.HasPrefix(s, "${") {
		if prop, present := properties[s[2:len(s)-1]]; !present {
			fmt.Printf("Failed property lookup %s: %s\n", s, properties)
			os.Exit(4)
		} else {
			return prop
		}
	}
	return s
}

// parse parses a downloaded pom.xml. This is of course less trivial than you would hope.
func parse(response []byte, group, artifact, version string) *pomXml {
	pom := &pomXml{}
	if err := xml.Unmarshal(response, pom); err != nil {
		log.Fatalf("Error parsing XML response: %s\n", err)
	} else if (pom.GroupId != "" && group != pom.GroupId) ||
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
	handleDependencies(pom.Dependencies, properties, version)
	handleDependencies(pom.DependencyManagement.Dependencies, properties, version)
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

func handleDependencies(deps pomDependencies, properties map[string]string, version string) {
	for _, dep := range deps.Dependency {
		// This is a bit of a hack; our build model doesn't distinguish these in the way Maven does.
		// TODO(pebers): Consider allowing specifying these to this tool to produce test-only deps.
		if dep.Scope == "test" {
			continue
		}
		dep.GroupId = replaceVariables(dep.GroupId, properties)
		dep.ArtifactId = replaceVariables(dep.ArtifactId, properties)
		dep.Version = replaceVariables(dep.Version, properties)
		if isExcluded(dep.ArtifactId) {
			continue
		}
		if dep.Version == "" {
			dep.Version = version
		}
		licences := strings.Join(fetchLicences(dep.GroupId, dep.ArtifactId, dep.Version), "|")
		if licences != "" {
			fmt.Printf("%s:%s:%s:%s\n", dep.GroupId, dep.ArtifactId, dep.Version, licences)
		} else {
			fmt.Printf("%s:%s:%s\n", dep.GroupId, dep.ArtifactId, dep.Version)
		}
		opts.Exclude = append(opts.Exclude, dep.ArtifactId) // Don't do this one again.
		// Recurse so we get all transitive dependencies
		pom := fetchAndParse(dep.GroupId, dep.ArtifactId, dep.Version)
		process(pom, dep.GroupId, dep.ArtifactId, dep.Version)
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
		version = fetchMetadata(group, artifact).Versioning.Release
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
	ret := &mavenMetadataXml{}
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
