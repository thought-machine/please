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
