package maven

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

const mavenJarTemplate = `
{{- if Needed .GroupID .ArtifactID -}}
maven_jar(
    name = '{{ .ArtifactID }}',
    id = '{{ .GroupID }}:{{ .ArtifactID }}:{{ .Version }}',
    hash = '',{{ if .Dependencies.Dependency }}
    deps = [
{{ range .Dependencies.Dependency }}        '{{ Dep .GroupID .ArtifactID }}',
{{ end }}    ],{{ end }}
)
{{- end -}}
`

// AllDependencies returns all the dependencies of these artifacts in a short format
// that we consume later. The format is vaguely akin to a Maven id, although we consider
// it an internal detail - it must agree between this and the maven_jars build rule that
// consumes it, but we don't hold it stable between different Please versions. The format is:
// group_id:artifact_id:version:{src|no_src}[:licence|licence|...]
//
// Alternatively if buildRules is true, it will return a series of maven_jar rules
// that could be pasted into a BUILD file.
func AllDependencies(f *Fetch, artifacts []Artifact, concurrency int, indent, buildRules bool, graph *Graph) []string {
	f.Resolver.Run(artifacts, concurrency)
	f.Resolver.Mediate()
	graph.BuildMapping()

	done := map[unversioned]bool{}
	ret := []string{}
	for _, a := range artifacts {
		ret = append(ret, allDeps(f.Pom(&a), indent, buildRules, done, graph)...)
	}
	return ret
}

func allDeps(pom *PomXML, indent, buildRules bool, done map[unversioned]bool, graph *Graph) []string {
	if buildRules {
		tmpl := template.Must(template.New("maven_jar").Funcs(template.FuncMap{
			"Dep":    graph.Dep,
			"Needed": graph.Needed,
		}).Parse(mavenJarTemplate))
		return allDependencies(pom, "", "", tmpl, done)
	}

	indentIncrement := ""
	if indent {
		indentIncrement = "  "
	}
	// Just run through dependencies here, not the top-level pom itself.
	ret := []string{}
	for _, dep := range pom.AllDependencies() {
		if !done[dep.unversioned] {
			done[dep.unversioned] = true
			ret = append(ret, allDependencies(dep, "", indentIncrement, nil, done)...)
		}
	}
	return ret
}

// allDependencies implements the logic of AllDependencies with indenting.
func allDependencies(pom *PomXML, currentIndent, indentIncrement string, tmpl *template.Template, done map[unversioned]bool) []string {
	ret := []string{}
	if tmpl != nil {
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, pom); err != nil {
			log.Fatalf("%s\n", err)
		} else if buf.Len() > 0 {
			ret = []string{buf.String()}
		}
	} else {
		ret = []string{fmt.Sprintf("%s%s:%s", currentIndent, pom.Artifact, source(pom))}
		if licences := pom.AllLicences(); len(licences) > 0 {
			ret[0] += ":" + strings.Join(licences, "|")
		}
	}
	for _, dep := range pom.AllDependencies() {
		if !done[dep.unversioned] {
			done[dep.unversioned] = true
			ret = append(ret, allDependencies(dep, currentIndent+indentIncrement, indentIncrement, tmpl, done)...)
		}
	}
	return ret
}

// source returns the src / no_src indicator for a single pom.
func source(pom *PomXML) string {
	if pom.HasSources {
		return "src"
	}
	return "no_src"
}
